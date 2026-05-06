package factory

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"dario.cat/mergo"
	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/repo"
	"github.com/ognev-dev/goplease/tracing"
)

// Factory holds dependencies required by factory methods.
type Factory struct {
	db   *app.DB
	repo *repo.Repo
}

// New is a factory function that creates and returns a new Factory instance.
func New(db *app.DB) *Factory {
	return &Factory{
		db:   db,
		repo: repo.New(db, tracing.NewNoOpTracer()),
	}
}

func merge(dst, src any) {
	err := mergo.Merge(dst, src, mergo.WithOverride, mergo.WithTransformers(mergeTransformer{}))
	if err != nil {
		panic(fmt.Sprintf("Unable to merge %T with %T", dst, src))
	}
}

type mergeTransformer struct{}

func (t mergeTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == reflect.TypeFor[time.Time]() {
		return func(dst, src reflect.Value) error {
			if src.CanInterface() {
				timeVal, ok := src.Interface().(time.Time)
				if ok && !timeVal.IsZero() {
					if dst.CanSet() {
						dst.Set(src)
					}
				}
			}
			return nil
		}
	}

	if typ == reflect.TypeFor[ds.ID]() {
		return func(dst, src reflect.Value) error {
			if src.CanInterface() {
				id, ok := src.Interface().(ds.ID)
				if ok && !id.IsNil() {
					if dst.CanSet() {
						dst.Set(src)
					}
				}
			}
			return nil
		}
	}

	return nil
}

// Batch is a function that repeatedly executes a data creation function ('createFn')
// a specified number of times and returns a slice of pointers to the created objects.
//
// T is the type of the struct being created (e.g., ds.User).
// fn is the function that creates a single instance of T (e.g., CreateUser).
// override allows passing custom field values to override defaults in the created instances.
func Batch[T any](size int, createFn func(m ...T) (*T, error), override ...T) ([]*T, error) {
	data := make([]*T, size)
	for i := range data {
		var err error
		data[i], err = createFn(override...)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

// Two is a convenience function to create exactly two instances of a data structure T.
// It is a wrapper around Batch with size=2.
func Two[T any](fn func(m ...T) (*T, error), override ...T) ([]*T, error) {
	return Batch(2, fn, override...) //nolint:mnd
}

// Five is a convenience function to create exactly five instances of a data structure T.
// It is a wrapper around Batch with size=5.
func Five[T any](fn func(m ...T) (*T, error), override ...T) ([]*T, error) {
	return Batch(5, fn, override...) //nolint:mnd
}

// Ten is a convenience function to create exactly ten instances of a data structure T.
// It is a wrapper around Batch with size=10.
func Ten[T any](fn func(m ...T) (*T, error), override ...T) ([]*T, error) {
	return Batch(10, fn, override...) //nolint:mnd
}

// LookupUnique tries to find a unique value in the given table.column by repeatedly
// querying the database with a transformed version of the input value.
//
// The transformFn is expected to produce a new candidate value
// (for example, by appending a suffix or incrementing a counter) and must eventually
// lead to a value that does not exist in the database, otherwise the function will
// recurse indefinitely (angry emoji).
func LookupUnique[T any](ctx context.Context, db *app.DB, table, column string, value T,
	transformFn func(T) T) (T, error) {
	res := new(T)

	q, _, err := sq.Select(column).
		From(table).
		Where(sq.Eq{column: value}).
		Limit(1).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return value, err
	}

	err = pgxscan.Get(ctx, db, res, q, value)
	if errors.Is(err, pgx.ErrNoRows) {
		return value, nil
	}
	if err != nil {
		return value, err
	}

	return LookupUnique[T](ctx, db, table, column, transformFn(value), transformFn)
}
