package service_test

import (
	"os"
	"testing"

	"github.com/ognev-dev/goplease/test"
)

var tt *test.App

func TestMain(m *testing.M) {
	tt = test.NewApp()

	code := m.Run()

	tt.Shutdown()
	os.Exit(code)
}

func create[T any](t *testing.T, override ...T) *T {
	t.Helper()

	return test.Create[T](t, tt.Factory, override...)
}
