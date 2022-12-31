package server

import (
	"context"
	"encoding/json"
	"fmt"
	nethttp "net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	age     = 24 * time.Hour
	counter = 2
	header  = 1 << 20
	timeout = 10 * time.Second
)

type Server interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Run(context.Context) error
}

type Config struct {
	Config   config.Config
	Logger   hclog.Logger
	Port     int
	Queue    queue.Queue
	Ssh      connect.Ssh
	Storage  storage.Storage
	Watchdog watchdog.Watchdog
}

type httpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type server struct {
	cfg    *Config
	engine *gin.Engine
}

func New(_ context.Context, cfg *Config) Server {
	return &server{
		cfg:    cfg,
		engine: nil,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (s *server) Init(ctx context.Context) error {
	s.cfg.Logger.Debug("log: Init")

	if err := s.cfg.Queue.Init(ctx); err != nil {
		return errors.Wrap(err, "failed to init queue")
	}

	if err := s.cfg.Ssh.Init(ctx); err != nil {
		return errors.Wrap(err, "failed to init ssh")
	}

	if err := s.cfg.Storage.Init(ctx); err != nil {
		return errors.Wrap(err, "failed to init storage")
	}

	if err := s.cfg.Watchdog.Init(ctx); err != nil {
		return errors.Wrap(err, "failed to init watchdog")
	}

	if err := s.initRoute(ctx); err != nil {
		return errors.Wrap(err, "failed to init route")
	}

	return nil
}

func (s *server) Deinit(ctx context.Context) error {
	s.cfg.Logger.Debug("log: Deinit")

	_ = s.cfg.Watchdog.Deinit(ctx)
	_ = s.cfg.Storage.Deinit(ctx)
	_ = s.cfg.Ssh.Deinit(ctx)
	_ = s.cfg.Queue.Deinit(ctx)

	return nil
}

func (s *server) Run(ctx context.Context) error {
	s.cfg.Logger.Debug("log: Run")

	var err error
	var wg sync.WaitGroup

	buf := make(chan string)

	go func(c context.Context, b chan string) {
		s.fetchEvent(c, b)
	}(ctx, buf)

	wg.Add(counter)

	go func(c context.Context, b chan string) {
		defer wg.Done()
		for item := range b {
			err = s.cfg.Queue.Put(c, item)
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
		err = s.storeEvent(ctx)
		if err != nil {
			return
		}
	}(ctx)

	wg.Wait()

	return err
}

func (s *server) initRoute(_ context.Context) error {
	handler := func(ctx *gin.Context) {
		var err error
		q := ctx.Request.URL.Query().Get("q")
		fmt.Printf("%v\n", q)
		if err != nil {
			ctx.JSON(nethttp.StatusNotFound, httpError{Code: nethttp.StatusNotFound, Message: err.Error()})
			return
		}

		ctx.JSON(nethttp.StatusOK, events.Event{})
	}

	s.engine = gin.New()
	if s.engine == nil {
		return errors.New("failed to create gin")
	}

	s.engine.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowHeaders:     []string{"*"},
		AllowMethods:     []string{"GET"},
		AllowOrigins:     []string{"*"},
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		ExposeHeaders: []string{"Content-Type"},
		MaxAge:        age,
	}))

	s.engine.Use(gin.Logger())
	s.engine.Use(gin.Recovery())

	e := s.engine.Group("/events")
	e.GET("/", handler)

	return nil
}

func (s *server) fetchEvent(ctx context.Context, param chan string) {
	s.cfg.Logger.Debug("log: fetchEvent")

	reconn := make(chan bool, 1)
	start := make(chan bool, 1)

	_ = s.cfg.Ssh.Start(ctx, "stream-events", param)

	go func(ctx context.Context, reconn, start chan bool) {
		_ = s.cfg.Watchdog.Run(ctx, s.cfg.Ssh, reconn, start)
	}(ctx, reconn, start)

	for {
		select {
		case <-reconn:
			if err := s.cfg.Ssh.Reconnect(ctx); err == nil {
				start <- true
			}
		case <-start:
			_ = s.cfg.Ssh.Start(ctx, "stream-events", param)
		}
	}
}

func (s *server) storeEvent(ctx context.Context) error {
	s.cfg.Logger.Debug("log: storeEvent")

	var err error
	var r chan string

	e := events.Event{}

	r, err = s.cfg.Queue.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get queue")
	}

	for item := range r {
		if err = json.Unmarshal([]byte(item), &e); err != nil {
			break
		}
		// TODO: storeEvent
	}

	return err
}

func (s *server) queryEvent(_ context.Context) error {
	s.cfg.Logger.Debug("log: queryEvent")

	var err error

	srv := &nethttp.Server{
		Addr:           ":" + strconv.Itoa(s.cfg.Port),
		Handler:        s.engine,
		ReadTimeout:    timeout,
		WriteTimeout:   timeout,
		MaxHeaderBytes: header,
	}

	go func() {
		err = srv.ListenAndServe()
		if err != nil {
			return
		}
	}()

	return err
}
