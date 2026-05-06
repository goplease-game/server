package worker_test

import (
	"context"
	"os"
	"testing"

	"github.com/ognev-dev/goplease/test"
	"github.com/ognev-dev/goplease/worker"
)

var tt *test.App

func TestMain(m *testing.M) {
	tt = test.NewApp()

	code := m.Run()

	tt.Shutdown()
	os.Exit(code)
}

func runJob(t *testing.T, j worker.Job) {
	t.Helper()
	t.Logf("RUNNING JOB: %s", j.Name())

	err := j.Do(context.Background(), tt.Service, tt.DB)
	test.CheckErr(t, err)
}

func create[T any](t *testing.T, override ...T) *T {
	t.Helper()

	return test.Create[T](t, tt.Factory, override...)
}
