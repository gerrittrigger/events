package server

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"

	"github.com/gerrittrigger/events/config"
	"github.com/gerrittrigger/events/storage"
)

const (
	name = "test.db"
)

var (
	data = []storage.Model{
		{
			EventBase64:    "ZXZlbnRCYXNlNjQ=",
			EventCreatedOn: 1672567200,
		},
	}
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

	s.cfg.Storage = initStorage()
	_ = s.cfg.Storage.Init(ctx)
	_ = s.cfg.Storage.Create(ctx, data)

	_ = s.initHttp(ctx)

	return s
}

func initStorage() storage.Storage {
	c := storage.DefaultConfig()
	ctx := context.Background()

	c.Config = config.Config{}
	c.Config.Spec.Storage.Sqlite.Filename = name

	c.Logger = hclog.New(&hclog.LoggerOptions{
		Name:  "storage",
		Level: hclog.LevelFromString("INFO"),
	})

	return storage.New(ctx, c)
}

func TestQueryEvent(t *testing.T) {
	s := initServer()

	rec := httptest.NewRecorder()
	req, _ := nethttp.NewRequest("GET", "/events/?q=", nethttp.NoBody)
	s.engine.ServeHTTP(rec, req)
	assert.NotEqual(t, nethttp.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req, _ = nethttp.NewRequest("GET", `/events/?q=since:2023-01-01+10:00:00+until:2023-01-01+11:00:00`, nethttp.NoBody)
	s.engine.ServeHTTP(rec, req)
	assert.Equal(t, nethttp.StatusOK, rec.Code)
	assert.NotEqual(t, nil, rec.Body.String())

	_ = os.Remove(name)
}

func TestParseQuery(t *testing.T) {
	s := initServer()

	_, _, err := s.parseQuery("")
	assert.NotEqual(t, nil, err)

	_, _, err = s.parseQuery("since: until:")
	assert.NotEqual(t, nil, err)

	_, _, err = s.parseQuery("2023-01-01 10:00:00 until:2023-01-01 11:00:00")
	assert.NotEqual(t, nil, err)

	_, _, err = s.parseQuery("since:2023-01-01 10:00:00 2023-01-01 11:00:00")
	assert.NotEqual(t, nil, err)

	_, _, err = s.parseQuery("since:2023-01-01 10:00:00 until:2023-01-01 11:00:00")
	assert.Equal(t, nil, err)

	_ = os.Remove(name)
}
