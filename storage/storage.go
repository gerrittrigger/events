package storage

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/gerrittrigger/events/config"
)

const (
	BatchSize  = 100
	PrimaryKey = "event_created_on"
)

type Storage interface {
	Init(context.Context) error
	Deinit(context.Context) error
	Create(context.Context, []Model) error
	Delete(context.Context, []int) error
	Read(context.Context, int, int) ([]Model, error)
	Update(context.Context, *Model) error
}

type Config struct {
	Config config.Config
	Logger hclog.Logger
}

type Model struct {
	EventBase64    string `json:"event_base64" gorm:"type:text"`
	EventCreatedOn int    `json:"event_created_on" gorm:"primarykey"`
}

type storage struct {
	cfg      *Config
	database *gorm.DB
}

func New(_ context.Context, cfg *Config) Storage {
	return &storage{
		cfg:      cfg,
		database: nil,
	}
}

func DefaultConfig() *Config {
	return &Config{}
}

func (s *storage) Init(ctx context.Context) error {
	s.cfg.Logger.Debug("storage: Init")

	var err error

	s.database, err = gorm.Open(sqlite.Open(s.cfg.Config.Spec.Storage.Sqlite.Filename), &gorm.Config{})
	if err != nil {
		return errors.Wrap(err, "failed to connect database")
	}

	if err = s.database.AutoMigrate(&Model{}); err != nil {
		_ = s.Deinit(ctx)
		return errors.Wrap(err, "failed to migrate database")
	}

	return nil
}

func (s *storage) Deinit(_ context.Context) error {
	s.cfg.Logger.Debug("storage: Deinit")

	if s.database == nil {
		return nil
	}

	if d, err := s.database.DB(); err == nil && d != nil {
		_ = d.Close()
	}

	return nil
}

func (s *storage) Create(_ context.Context, data []Model) error {
	s.cfg.Logger.Debug("storage: Create")

	if len(data) == 0 || len(data) > BatchSize {
		return errors.New("invalid data length")
	}

	if r := s.database.CreateInBatches(data, BatchSize); r.Error != nil {
		return errors.Wrap(r.Error, "failed to create")
	}

	return nil
}

func (s *storage) Delete(_ context.Context, key []int) error {
	s.cfg.Logger.Debug("storage: Delete")

	var b []Model

	if len(key) == 0 {
		return errors.New("invalid key length")
	}

	if r := s.database.Delete(&b, key); r.Error != nil {
		return errors.Wrap(r.Error, "failed to delete")
	}

	return nil
}

func (s *storage) Read(_ context.Context, since, until int) ([]Model, error) {
	s.cfg.Logger.Debug("storage: Read")

	var b []Model

	if since <= 0 || until <= 0 {
		return nil, errors.New("invalid date")
	}

	r := s.database.Where(fmt.Sprintf("%s >= ? AND %s < ?", PrimaryKey, PrimaryKey), since, until).Find(&b)
	if r.Error != nil {
		return nil, errors.Wrap(r.Error, "failed to read")
	}

	return b, nil
}

func (s *storage) Update(_ context.Context, data *Model) error {
	s.cfg.Logger.Debug("storage: Update")

	var b Model

	if data == nil {
		return errors.New("invalid data")
	}

	r := s.database.Model(&b).Where(fmt.Sprintf("%s = ?", PrimaryKey), data.EventCreatedOn).Updates(data)
	if r.Error != nil {
		return errors.Wrap(r.Error, "failed to update")
	}

	return nil
}
