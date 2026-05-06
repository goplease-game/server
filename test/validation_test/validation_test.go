package validation_test

import (
	"errors"
	"testing"

	"github.com/ognev-dev/goplease/app"
	"github.com/stretchr/testify/assert"
)

func checkValidatedInput(t *testing.T, valid bool, err error, argName string, expectedErr string) {
	t.Helper()

	if valid && err != nil {
		t.Error(err)
	}

	if valid {
		return
	}

	if err == nil {
		t.Fatal("expected error")
	}

	inputError, ok := errors.AsType[app.InputError](err)
	if !ok {
		t.Error("[ERROR]: ", err.Error())
		t.Fatalf("error expected to be of type InputError, %T given", err)
	}

	errValue, ok := inputError[argName]
	if !ok {
		t.Fatalf("failed to find key '%s' in error message", argName)
	}

	assert.Equal(t, expectedErr, errValue)
}
