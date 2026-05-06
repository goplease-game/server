package service_test

import (
	"context"
	"testing"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/test"
)

func TestFilterUsers(t *testing.T) {
	create[ds.User](t)

	_, _, err := tt.Service.FilterUsers(context.Background(), ds.UsersFilter{})
	test.CheckErr(t, err)
}
