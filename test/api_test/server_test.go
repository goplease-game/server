package api_test

import (
	"testing"

	"github.com/ognev-dev/goplease/server/response"
	"github.com/stretchr/testify/assert"
)

func TestGetServerStatus(t *testing.T) {
	var resp response.ServerStatus
	GET(t, "status", &resp)

	assert.Equal(t, resp.Env, tt.Conf.App.Env)
	assert.Equal(t, resp.Version, tt.Conf.App.Version)
}
