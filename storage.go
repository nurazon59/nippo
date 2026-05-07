package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adrg/xdg"
)

type Storage struct {
	baseDir string
}

func NewStorage(storageDir string) (*Storage, error) {
	dir := storageDir
	if dir == "" {
		dir = xdg.DataHome
	}
	return &Storage{baseDir: dir}, nil
}

func (s *Storage) reportDir(date time.Time) string {
	return filepath.Join(s.baseDir, "nippo", date.Format("2006/01"))
}

func (s *Storage) reportPath(date time.Time) string {
	return filepath.Join(s.reportDir(date), date.Format("02")+".md")
}

func (s *Storage) SaveReport(content string, date time.Time) error {
	dir := s.reportDir(date)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(s.reportPath(date), []byte(content), 0644)
}

func (s *Storage) LoadReport(date time.Time) (string, error) {
	bytes, err := os.ReadFile(s.reportPath(date))
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (s *Storage) ListReports() ([]string, error) {
	base := filepath.Join(s.baseDir, "nippo")
	var reports []string

	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		reports = append(reports, rel)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	sort.Sort(sort.Reverse(sort.StringSlice(reports)))
	return reports, nil
}
