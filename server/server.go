package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	nethttp "net/http"
	"strconv"
	"strings"
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
	queryLayout = "2006-01-02 15:04:05"
	queryLength = 2
	querySince  = "since:"
	queryUntil  = "until:"

	maxAge      = 24 * time.Hour
	maxDuration = 10 * time.Second
	maxHeader   = 1 << 20
	waitCount   = 2
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

type httpResult struct {
	EventBase64    string `json:"eventBase64"`
	EventCreatedOn int64  `json:"eventCreatedOn"`
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
	s.cfg.Logger.Debug("server: Init")

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

	if err := s.initHttp(ctx); err != nil {
		return errors.Wrap(err, "failed to init http")
	}

	if err := s.listenHttp(ctx); err != nil {
		return errors.Wrap(err, "failed to listen http")
	}

	return nil
}

func (s *server) Deinit(ctx context.Context) error {
	s.cfg.Logger.Debug("server: Deinit")

	_ = s.cfg.Watchdog.Deinit(ctx)
	_ = s.cfg.Storage.Deinit(ctx)
	_ = s.cfg.Ssh.Deinit(ctx)
	_ = s.cfg.Queue.Deinit(ctx)

	return nil
}

func (s *server) Run(ctx context.Context) error {
	s.cfg.Logger.Debug("server: Run")

	var err error
	var wg sync.WaitGroup

	buf := make(chan string)

	go func(c context.Context, b chan string) {
		s.fetchEvent(c, b)
	}(ctx, buf)

	wg.Add(waitCount)

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

func (s *server) initHttp(_ context.Context) error {
	s.cfg.Logger.Debug("server: initHttp")

	handler := func(ctx *gin.Context) {
		q := ctx.Request.URL.Query().Get("q")
		b, err := s.queryEvent(ctx, q)
		if err != nil {
			ctx.JSON(nethttp.StatusNotFound, httpError{Code: nethttp.StatusNotFound, Message: err.Error()})
			return
		}
		ctx.JSON(nethttp.StatusOK, b)
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
		MaxAge:        maxAge,
	}))

	s.engine.Use(gin.Logger())
	s.engine.Use(gin.Recovery())

	e := s.engine.Group("/events")
	e.GET("/", handler)

	return nil
}

func (s *server) listenHttp(_ context.Context) error {
	s.cfg.Logger.Debug("server: listenHttp")

	var err error

	srv := &nethttp.Server{
		Addr:           ":" + strconv.Itoa(s.cfg.Port),
		Handler:        s.engine,
		ReadTimeout:    maxDuration,
		WriteTimeout:   maxDuration,
		MaxHeaderBytes: maxHeader,
	}

	go func() {
		err = srv.ListenAndServe()
		if err != nil {
			return
		}
	}()

	return err
}

func (s *server) fetchEvent(ctx context.Context, param chan string) {
	s.cfg.Logger.Debug("server: fetchEvent")

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

func (s *server) queryEvent(ctx context.Context, query string) ([]httpResult, error) {
	s.cfg.Logger.Debug("server: queryEvent")

	rs, ru, err := s.parseQuery(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse")
	}

	b, err := s.cfg.Storage.Read(ctx, rs, ru)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}

	if len(b) == 0 {
		return []httpResult{}, nil
	}

	m := make([]httpResult, len(b))

	for i := range b {
		m[i] = httpResult{
			EventBase64:    b[i].EventBase64,
			EventCreatedOn: b[i].EventCreatedOn,
		}
	}

	return m, nil
}

func (s *server) parseQuery(query string) (rs, ru int64, err error) {
	helper := func(q string) (int64, error) {
		t, e := time.Parse(queryLayout, q)
		if e != nil {
			return 0, errors.Wrap(e, "invalid format")
		}
		return t.Unix(), nil
	}

	if !strings.HasPrefix(query, querySince) || !strings.Contains(query, queryUntil) {
		return 0, 0, errors.New("missing query")
	}

	b := strings.Split(query, queryUntil)
	if len(b) != queryLength {
		return 0, 0, errors.New("invalid length")
	}

	_s := strings.Trim(strings.TrimPrefix(b[0], querySince), " ")
	u := strings.Trim(b[1], " ")

	if _s == "" || u == "" {
		return 0, 0, errors.New("empty query")
	}

	rs, err = helper(_s)
	if err != nil {
		return 0, 0, errors.New("invalid query")
	}

	ru, err = helper(u)
	if err != nil {
		return 0, 0, errors.New("invalid query")
	}

	return rs, ru, nil
}

func (s *server) storeEvent(ctx context.Context) error {
	s.cfg.Logger.Debug("server: storeEvent")

	var err error
	var r chan string

	r, err = s.cfg.Queue.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get queue")
	}

	for item := range r {
		e := events.Event{}
		if err = json.Unmarshal([]byte(item), &e); err != nil {
			break
		}
		b := []storage.Model{{EventBase64: base64.StdEncoding.EncodeToString([]byte(item)), EventCreatedOn: e.EventCreatedOn}}
		if err = s.cfg.Storage.Create(ctx, b); err != nil {
			break
		}
	}

	return err
}
