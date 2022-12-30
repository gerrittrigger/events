package connect

import (
	"context"
	"fmt"
	nethttp "net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"

	"github.com/gerrittrigger/events/config"
	"github.com/gerrittrigger/events/events"
)

const (
	age     = 24 * time.Hour
	header  = 1 << 20
	timeout = 10 * time.Second
)

type Http interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Run(context.Context) error
}

type HttpConfig struct {
	Config config.Config
	Logger hclog.Logger
	Port   int
}

type http struct {
	cfg    *HttpConfig
	engine *gin.Engine
}

type httpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func HttpNew(_ context.Context, cfg *HttpConfig) Http {
	return &http{
		cfg:    cfg,
		engine: nil,
	}
}

func DefaultHttpConfig() *HttpConfig {
	return &HttpConfig{}
}

func (h *http) Init(ctx context.Context) error {
	h.cfg.Logger.Debug("http: Init")

	if err := h.initRoute(ctx); err != nil {
		return errors.Wrap(err, "failed to init route")
	}

	if err := h.setRoute(ctx); err != nil {
		return errors.Wrap(err, "failed to set route")
	}

	return nil
}

func (h *http) Deinit(_ context.Context) error {
	h.cfg.Logger.Debug("http: Deinit")

	return nil
}

func (h *http) Run(_ context.Context) error {
	h.cfg.Logger.Debug("http: Run")

	var err error

	srv := &nethttp.Server{
		Addr:           ":" + strconv.Itoa(h.cfg.Port),
		Handler:        h.engine,
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

func (h *http) initRoute(_ context.Context) error {
	h.engine = gin.New()
	if h.engine == nil {
		return errors.New("failed to create gin")
	}

	h.engine.Use(cors.New(cors.Config{
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

	h.engine.Use(gin.Logger())
	h.engine.Use(gin.Recovery())

	return nil
}

func (h *http) setRoute(_ context.Context) error {
	e := h.engine.Group("/events")
	e.GET("/", h.queryEvents)

	return nil
}

func (h *http) queryEvents(ctx *gin.Context) {
	var err error

	q := ctx.Request.URL.Query().Get("q")
	fmt.Printf("%v\n", q)

	if err != nil {
		h.errorWrap(ctx, nethttp.StatusNotFound, err)
		return
	}

	ctx.JSON(nethttp.StatusOK, events.Event{})
}

func (h *http) errorWrap(ctx *gin.Context, status int, err error) {
	e := httpError{
		Code:    status,
		Message: err.Error(),
	}

	ctx.JSON(status, e)
}
