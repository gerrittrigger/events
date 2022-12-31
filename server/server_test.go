package server

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"

	"github.com/gerrittrigger/events/config"
)

func initServer() server {
	ctx := context.Background()

	s := server{
		cfg:    DefaultConfig(),
		engine: nil,
	}

	s.cfg.Config = config.Config{}

	s.cfg.Logger = hclog.New(&hclog.LoggerOptions{
		Name:  "server",
		Level: hclog.LevelFromString("INFO"),
	})

	s.cfg.Port = 8080

	_ = s.initRoute(ctx)

	return s
}

func TestQueryEvents(t *testing.T) {
	h := initServer()

	rec := httptest.NewRecorder()
	req, _ := nethttp.NewRequest("GET", "/events/?q=since:2023-01-01+until:2023-02-01", nethttp.NoBody)
	h.engine.ServeHTTP(rec, req)

	assert.Equal(t, nethttp.StatusOK, rec.Code)
	assert.NotEqual(t, nil, rec.Body.String())

	req, _ = nethttp.NewRequest("GET", `/events/?q=since:"2023-01-01 10:00:00"+until:"2023-01-01 11:00:00"`, nethttp.NoBody)
	h.engine.ServeHTTP(rec, req)

	assert.Equal(t, nethttp.StatusOK, rec.Code)
	assert.NotEqual(t, nil, rec.Body.String())
}
