package log

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"

	"github.com/gerrittrigger/events/config"
	"github.com/gerrittrigger/events/connect"
	"github.com/gerrittrigger/events/events"
	"github.com/gerrittrigger/events/queue"
	"github.com/gerrittrigger/events/storage"
	"github.com/gerrittrigger/events/watchdog"
)

const (
	counter = 2
)

type Log interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Run(context.Context) error
}

type Config struct {
	Config   config.Config
	Http     connect.Http
	Logger   hclog.Logger
	Queue    queue.Queue
	Ssh      connect.Ssh
	Storage  storage.Storage
	Watchdog watchdog.Watchdog
}

type log struct {
	cfg *Config
}

func New(_ context.Context, cfg *Config) Log {
	return &log{
		cfg: cfg,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (l *log) Init(ctx context.Context) error {
	l.cfg.Logger.Debug("log: Init")

	if err := l.cfg.Http.Init(ctx); err != nil {
		return errors.Wrap(err, "failed to init http")
	}

	if err := l.cfg.Queue.Init(ctx); err != nil {
		return errors.Wrap(err, "failed to init queue")
	}

	if err := l.cfg.Ssh.Init(ctx); err != nil {
		return errors.Wrap(err, "failed to init ssh")
	}

	if err := l.cfg.Storage.Init(ctx); err != nil {
		return errors.Wrap(err, "failed to init storage")
	}

	if err := l.cfg.Watchdog.Init(ctx); err != nil {
		return errors.Wrap(err, "failed to init watchdog")
	}

	return nil
}

func (l *log) Deinit(ctx context.Context) error {
	l.cfg.Logger.Debug("log: Deinit")

	_ = l.cfg.Watchdog.Deinit(ctx)
	_ = l.cfg.Storage.Deinit(ctx)
	_ = l.cfg.Ssh.Deinit(ctx)
	_ = l.cfg.Queue.Deinit(ctx)
	_ = l.cfg.Http.Deinit(ctx)

	return nil
}

func (l *log) Run(ctx context.Context) error {
	l.cfg.Logger.Debug("log: Run")

	var err error
	var wg sync.WaitGroup

	buf := make(chan string)

	go func(c context.Context, b chan string) {
		l.fetchEvent(c, b)
	}(ctx, buf)

	wg.Add(counter)

	go func(c context.Context, b chan string) {
		defer wg.Done()
		for item := range b {
			err = l.cfg.Queue.Put(c, item)
			if err != nil {
				return
			}
		}
	}(ctx, buf)

	if err != nil {
		return errors.Wrap(err, "failed to put queue")
	}

	go func(ctx context.Context) {
		defer wg.Done()
		err = l.postEvent(ctx)
		if err != nil {
			return
		}
	}(ctx)

	wg.Wait()

	return err
}

func (l *log) fetchEvent(ctx context.Context, param chan string) {
	l.cfg.Logger.Debug("log: fetchEvent")

	reconn := make(chan bool, 1)
	start := make(chan bool, 1)

	_ = l.cfg.Ssh.Start(ctx, "stream-events", param)

	go func(ctx context.Context, reconn, start chan bool) {
		_ = l.cfg.Watchdog.Run(ctx, l.cfg.Ssh, reconn, start)
	}(ctx, reconn, start)

	for {
		select {
		case <-reconn:
			if err := l.cfg.Ssh.Reconnect(ctx); err == nil {
				start <- true
			}
		case <-start:
			_ = l.cfg.Ssh.Start(ctx, "stream-events", param)
		}
	}
}

func (l *log) postEvent(ctx context.Context) error {
	l.cfg.Logger.Debug("log: postEvent")

	var err error
	var r chan string

	e := events.Event{}

	r, err = l.cfg.Queue.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get queue")
	}

	for item := range r {
		if err = json.Unmarshal([]byte(item), &e); err != nil {
			break
		}
		// TODO: postEvent
	}

	return err
}
