package main

import (
	"time"

	"github.com/adrg/xdg"
	"github.com/nurazon59/nippo/backends"
)

type Storage struct {
	backend backends.ReportStorage
}

func NewStorage(cfg *Config) (*Storage, error) {
	fallback := cfg.StorageDir
	if fallback == "" {
		fallback = xdg.DataHome
	}
	backend, err := backends.Build(cfg.Storage, fallback)
	if err != nil {
		return nil, err
	}
	return &Storage{backend: backend}, nil
}

func (s *Storage) SaveReport(content string, date time.Time) error {
	return s.backend.Save(content, date)
}

func (s *Storage) LoadReport(date time.Time) (string, error) {
	return s.backend.LoadReport(date)
}

func (s *Storage) LoadPreviousReport(date time.Time) (time.Time, error) {
	return s.backend.LoadPreviousReport(date)
}

func (s *Storage) ListReports() ([]string, error) {
	return s.backend.ListReports()
}

func (s *Storage) Close() error {
	return s.backend.Close()
}
