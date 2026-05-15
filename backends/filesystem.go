package backends

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adrg/xdg"
)

type FilesystemBackend struct {
	baseDir string
}

var _ ReportStorage = (*FilesystemBackend)(nil)

func NewFilesystemBackend(storageDir string) (*FilesystemBackend, error) {
	dir := storageDir
	if dir == "" {
		dir = xdg.DataHome
	}
	return &FilesystemBackend{baseDir: dir}, nil
}

func (s *FilesystemBackend) reportDir(date time.Time) string {
	return filepath.Join(s.baseDir, "nippo", date.Format("2006/01"))
}

func (s *FilesystemBackend) reportPath(date time.Time) string {
	return filepath.Join(s.reportDir(date), date.Format("02")+".md")
}

func (s *FilesystemBackend) SaveReport(content string, date time.Time) error {
	dir := s.reportDir(date)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(s.reportPath(date), []byte(content), 0644)
}

func (s *FilesystemBackend) LoadReport(date time.Time) (string, error) {
	bytes, err := os.ReadFile(s.reportPath(date))
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func normalizeReportDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func parseReportDate(path string) (time.Time, error) {
	normalized := filepath.ToSlash(path)
	return time.Parse("2006/01/02.md", normalized)
}

func (s *FilesystemBackend) LoadPreviousReport(date time.Time) (string, error) {
	reports, err := s.ListReports()
	if err != nil {
		return "", err
	}

	target := normalizeReportDate(date)
	for _, rel := range reports {
		reportDate, err := parseReportDate(rel)
		if err != nil {
			continue
		}
		if !reportDate.Before(target) {
			continue
		}

		bytes, err := os.ReadFile(filepath.Join(s.baseDir, "nippo", rel))
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}

	return "", fs.ErrNotExist
}

func (s *FilesystemBackend) ListReports() ([]string, error) {
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
