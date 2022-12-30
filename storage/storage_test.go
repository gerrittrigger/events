package storage

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

const (
	name = "test.db"
)

var (
	data = []Model{
		{
			EventBase64:    "ZXZlbnRCYXNlNjQ=",
			EventCreatedOn: 1672214667,
		},
	}
)

func initStorage() *storage {
	ctx := context.Background()

	s := &storage{
		cfg:      DefaultConfig(),
		database: nil,
	}

	s.cfg.Config.Spec.Storage.Sqlite.Filename = name

	s.cfg.Logger = hclog.New(&hclog.LoggerOptions{
		Name:  "storage",
		Level: hclog.LevelFromString("INFO"),
	})

	_ = s.Init(ctx)

	return s
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	s := initStorage()

	var b []Model

	err := s.Create(ctx, b)
	assert.NotEqual(t, nil, err)

	b = make([]Model, BatchSize+1)

	err = s.Create(ctx, b)
	assert.NotEqual(t, nil, err)

	err = s.Create(ctx, data)
	assert.Equal(t, nil, err)

	_ = os.Remove(name)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	s := initStorage()

	var b []int

	_ = s.Create(ctx, data)

	err := s.Delete(ctx, b)
	assert.NotEqual(t, nil, err)

	b = []int{data[0].EventCreatedOn}

	err = s.Delete(ctx, b)
	assert.Equal(t, nil, err)

	_ = os.Remove(name)
}

func TestRead(t *testing.T) {
	ctx := context.Background()
	s := initStorage()

	_ = s.Create(ctx, data)

	_, err := s.Read(ctx, 0, 0)
	assert.NotEqual(t, nil, err)

	b, err := s.Read(ctx, 1, data[0].EventCreatedOn+1)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(b))
	assert.Equal(t, data[0].EventBase64, b[0].EventBase64)

	_ = os.Remove(name)
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	s := initStorage()

	_ = s.Create(ctx, data)

	err := s.Update(ctx, nil)
	assert.NotEqual(t, nil, err)

	data[0].EventBase64 = "updated"

	err = s.Update(ctx, &data[0])
	assert.Equal(t, nil, err)

	b, err := s.Read(ctx, 1, data[0].EventCreatedOn+1)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(b))
	assert.Equal(t, data[0].EventBase64, b[0].EventBase64)

	_ = os.Remove(name)
}
