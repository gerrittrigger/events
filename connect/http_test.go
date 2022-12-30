package connect

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func initHttp() *http {
	ctx := context.Background()

	h := &http{
		cfg:    DefaultHttpConfig(),
		engine: nil,
	}

	h.cfg.Logger = hclog.New(&hclog.LoggerOptions{
		Name:  "http",
		Level: hclog.LevelFromString("INFO"),
	})

	h.cfg.Port = 8080

	_ = h.initRoute(ctx)
	_ = h.setRoute(ctx)

	return h
}

func TestQueryEvents(t *testing.T) {
	h := initHttp()

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
