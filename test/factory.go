//nolint:mnd
package test

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/ognev-dev/goplease/test/factory"
)

var (
	// ErrNoFactoryMethod is returned when no suitable factory method
	// can be found for the requested type.
	ErrNoFactoryMethod = errors.New("no factory method found")

	// ErrAmbiguousFactoryMethod is returned when multiple factory methods
	// match the requested type and the selection is ambiguous.
	ErrAmbiguousFactoryMethod = errors.New("ambiguous factory method")
)

var (
	// createMethodCache caches resolved factory methods by model type T.
	// Key: reflect.Type of T, Value: reflect.Method that returns *T.
	createMethodCache sync.Map

	// errorType is the reflect.Type of the built-in error interface.
	errorType = reflect.TypeFor[*error]().Elem()
)

// createMethodFor resolves and caches a factory.Factory "create" method.
func createMethodFor[T any]() (reflect.Method, error) {
	tType := reflect.TypeFor[*T]().Elem()

	if v, ok := createMethodCache.Load(tType); ok {
		return v.(reflect.Method), nil //nolint:forcetypeassert
	}

	factoryType := reflect.TypeFor[*factory.Factory]()
	wantOut0 := reflect.PointerTo(tType)

	var found *reflect.Method

	for m := range factoryType.Methods() {
		mt := m.Type

		// func(*Factory, ...T) (*T, error)
		if mt.NumOut() != 2 {
			continue
		}
		if mt.Out(0) != wantOut0 {
			continue
		}
		if !mt.Out(1).Implements(errorType) {
			continue
		}
		if !mt.IsVariadic() {
			continue
		}
		if mt.NumIn() != 2 {
			continue
		}
		// variadic arg is []T in reflect
		if mt.In(1) != reflect.SliceOf(tType) {
			continue
		}

		if found != nil {
			return reflect.Method{}, fmt.Errorf("%w for %v: %s and %s", ErrAmbiguousFactoryMethod, tType, found.Name, m.Name)
		}

		tmp := m
		found = &tmp
	}

	if found == nil {
		return reflect.Method{}, fmt.Errorf("%w for *%v", ErrNoFactoryMethod, tType)
	}

	createMethodCache.Store(tType, *found)
	return *found, nil
}

// Create invokes the corresponding Create{Model} method on factory.Factory,
// where Model is inferred from T.
//
// Optional overrideOpt values are forwarded as variadic arguments to the
// underlying factory method. The test fails immediately on any error.
//
// NOTE: Go does not currently allow methods to declare their own type parameters.
// When this becomes possible, move this helper onto *factory.Factory
// and remove the wrapper functions.
// TODO: a promising "Proposal: Generic Methods for Go" have landed:  https://github.com/golang/go/issues/77273
func Create[T any](t *testing.T, f *factory.Factory, overrideOpt ...T) *T {
	t.Helper()

	if f == nil {
		t.Fatal("Create.Factory is nil")
	}

	m, err := createMethodFor[T]()
	if err != nil {
		t.Fatal(err)
	}

	args := make([]reflect.Value, 1, 2)
	args[0] = reflect.ValueOf(f)
	if len(overrideOpt) > 0 {
		args = append(args, reflect.ValueOf(overrideOpt[0]))
	}

	outs := m.Func.Call(args)
	if !outs[1].IsNil() {
		err := outs[1].Interface().(error) //nolint:forcetypeassert
		t.Fatalf("[factory.%s]: %s", m.Name, err)
	}

	return outs[0].Interface().(*T) //nolint:forcetypeassert
}
