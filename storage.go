package main

import (
	"time"

	"github.com/adrg/xdg"
	"github.com/nurazon59/nippo/backends"
)

type ReportStorage = backends.ReportStorage

type Storage struct {
	backend *backends.FilesystemBackend
}

func NewStorage(storageDir string) (*Storage, error) {
	dir := storageDir
	if dir == "" {
		dir = xdg.DataHome
	}
	backend, err := backends.NewFilesystemBackend(dir)
	if err != nil {
		return nil, err
	}
	return &Storage{backend: backend}, nil
}

func (s *Storage) Save(content string, date time.Time) error {
	return s.backend.Save(content, date)
}

func (s *Storage) LoadReport(date time.Time) (string, error) {
	return s.backend.LoadReport(date)
}

func (s *Storage) LoadPreviousReport(date time.Time) (string, error) {
	return s.backend.LoadPreviousReport(date)
}

func (s *Storage) ListReports() ([]string, error) {
	return s.backend.ListReports()
}
