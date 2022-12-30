package cmd

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/gerrittrigger/events/config"
)

const (
	port = 8080
)

func testInitConfig() *config.Config {
	cfg := config.New()

	fi, _ := os.Open("../test/config/config.yml")

	defer func() {
		_ = fi.Close()
	}()

	buf, _ := io.ReadAll(fi)
	_ = yaml.Unmarshal(buf, cfg)

	return cfg
}

func TestInitLogger(t *testing.T) {
	ctx := context.Background()

	logger, err := initLogger(ctx, level)
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, logger)
}

func TestInitConfig(t *testing.T) {
	logger, _ := initLogger(context.Background(), level)
	ctx := context.Background()

	_, err := initConfig(ctx, logger, "invalid.yml")
	assert.NotEqual(t, nil, err)

	_, err = initConfig(ctx, logger, "../test/config/invalid.yml")
	assert.NotEqual(t, nil, err)

	_, err = initConfig(ctx, logger, "../test/config/config.yml")
	assert.Equal(t, nil, err)
}

func TestInitConnect(t *testing.T) {
	logger, _ := initLogger(context.Background(), level)
	cfg := testInitConfig()

	_, _, err := initConnect(context.Background(), logger, cfg, port)
	assert.Equal(t, nil, err)
}

func TestInitQueue(t *testing.T) {
	logger, _ := initLogger(context.Background(), level)
	cfg := testInitConfig()

	_, err := initQueue(context.Background(), logger, cfg)
	assert.Equal(t, nil, err)
}

func TestInitStorage(t *testing.T) {
	logger, _ := initLogger(context.Background(), level)
	cfg := testInitConfig()

	_, err := initStorage(context.Background(), logger, cfg)
	assert.Equal(t, nil, err)
}

func TestInitWatchdog(t *testing.T) {
	logger, _ := initLogger(context.Background(), level)
	cfg := testInitConfig()

	_, err := initWatchdog(context.Background(), logger, cfg)
	assert.Equal(t, nil, err)
}

func TestInitServer(t *testing.T) {
	logger, _ := initLogger(context.Background(), level)
	cfg := testInitConfig()

	_, err := initServer(context.Background(), logger, cfg, port, nil, nil, nil)
	assert.Equal(t, nil, err)
}
