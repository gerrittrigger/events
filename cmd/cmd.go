package cmd

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v3"

	"github.com/gerrittrigger/events/config"
	"github.com/gerrittrigger/events/connect"
	"github.com/gerrittrigger/events/log"
	"github.com/gerrittrigger/events/queue"
	"github.com/gerrittrigger/events/storage"
	"github.com/gerrittrigger/events/watchdog"
)

const (
	level = "INFO"
	name  = "events"
)

var (
	app        = kingpin.New(name, "gerrit events").Version(config.Version + "-build-" + config.Build)
	configFile = app.Flag("config-file", "Config file (.yml)").Required().String()
	listenPort = app.Flag("listen-port", "Listen port").Default("8080").Int()
	logLevel   = app.Flag("log-level", "Log level (DEBUG|INFO|WARN|ERROR)").Default(level).String()
)

func Run(ctx context.Context) error {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	logger, err := initLogger(ctx, *logLevel)
	if err != nil {
		return errors.Wrap(err, "failed to init logger")
	}

	cfg, err := initConfig(ctx, logger, *configFile)
	if err != nil {
		return errors.Wrap(err, "failed to init config")
	}

	mq, err := initQueue(ctx, logger, cfg)
	if err != nil {
		return errors.Wrap(err, "failed to init queue")
	}

	st, err := initStorage(ctx, logger, cfg)
	if err != nil {
		return errors.Wrap(err, "failed to init storage")
	}

	wd, err := initWatchdog(ctx, logger, cfg)
	if err != nil {
		return errors.Wrap(err, "failed to init watchdog")
	}

	l, err := initLog(ctx, logger, cfg, *listenPort, mq, st, wd)
	if err != nil {
		return errors.Wrap(err, "failed to init log")
	}

	if err := runLog(ctx, logger, l); err != nil {
		return errors.Wrap(err, "failed to run log")
	}

	return nil
}

func initLogger(_ context.Context, level string) (hclog.Logger, error) {
	return hclog.New(&hclog.LoggerOptions{
		Name:  name,
		Level: hclog.LevelFromString(level),
	}), nil
}

func initConfig(_ context.Context, logger hclog.Logger, name string) (*config.Config, error) {
	logger.Debug("cmd: initConfig")

	c := config.New()

	fi, err := os.Open(name)
	if err != nil {
		return c, errors.Wrap(err, "failed to open")
	}

	defer func() {
		_ = fi.Close()
	}()

	buf, _ := io.ReadAll(fi)

	if err := yaml.Unmarshal(buf, c); err != nil {
		return c, errors.Wrap(err, "failed to unmarshal")
	}

	return c, nil
}

func initConnect(ctx context.Context, logger hclog.Logger, cfg *config.Config, port int) (connect.Http, connect.Ssh, error) {
	logger.Debug("cmd: initConnect")

	hc := connect.DefaultHttpConfig()
	if hc == nil {
		return nil, nil, errors.New("failed to config http")
	}

	hc.Config = *cfg
	hc.Logger = logger
	hc.Port = port

	sc := connect.DefaultSshConfig()
	if sc == nil {
		return nil, nil, errors.New("failed to config ssh")
	}

	sc.Config = *cfg
	sc.Logger = logger

	return connect.HttpNew(ctx, hc), connect.SshNew(ctx, sc), nil
}

func initQueue(ctx context.Context, logger hclog.Logger, cfg *config.Config) (queue.Queue, error) {
	logger.Debug("cmd: initQueue")

	c := queue.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	c.Config = *cfg
	c.Logger = logger

	return queue.New(ctx, c), nil
}

func initStorage(ctx context.Context, logger hclog.Logger, cfg *config.Config) (storage.Storage, error) {
	logger.Debug("cmd: initStorage")

	c := storage.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	c.Config = *cfg
	c.Logger = logger

	return storage.New(ctx, c), nil
}

func initWatchdog(ctx context.Context, logger hclog.Logger, cfg *config.Config) (watchdog.Watchdog, error) {
	logger.Debug("cmd: initWatchdog")

	c := watchdog.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	c.Config = *cfg
	c.Logger = logger

	return watchdog.New(ctx, c), nil
}

func initLog(ctx context.Context, logger hclog.Logger, cfg *config.Config, port int, mq queue.Queue, st storage.Storage,
	wd watchdog.Watchdog) (log.Log, error) {
	logger.Debug("cmd: initLog")

	var err error

	c := log.DefaultConfig()
	if c == nil {
		return nil, errors.New("failed to config")
	}

	c.Config = *cfg
	c.Logger = logger
	c.Queue = mq
	c.Storage = st
	c.Watchdog = wd

	c.Http, c.Ssh, err = initConnect(ctx, logger, cfg, port)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init connect")
	}

	return log.New(ctx, c), nil
}

func runLog(ctx context.Context, logger hclog.Logger, l log.Log) error {
	logger.Debug("cmd: runLog")

	if err := l.Init(ctx); err != nil {
		return errors.Wrap(err, "failed to init")
	}

	s := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can"t be caught, so don't need add it
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)

	go func(c context.Context, l log.Log) {
		logger.Debug("cmd: runLog: Run")
		_ = l.Run(c)
	}(ctx, l)

	go func(c context.Context, l log.Log, s chan os.Signal) {
		logger.Debug("cmd: runLog: Deinit")
		<-s
		_ = l.Deinit(c)
		done <- true
	}(ctx, l, s)

	<-done

	return nil
}
