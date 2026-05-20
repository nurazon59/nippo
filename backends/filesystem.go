package backends

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ReportStorage interface {
	Save(content string, date time.Time) error
	LoadReport(date time.Time) (string, error)
	LoadPreviousReport(date time.Time) (time.Time, error)
	LoadLatestReport() (time.Time, error)
	ListReports() ([]string, error)
	Close() error
}

type FilesystemBackend struct {
	baseDir string
}

func NewFilesystemBackend(baseDir string) *FilesystemBackend {
	return &FilesystemBackend{baseDir: baseDir}
}

func (b *FilesystemBackend) reportDir(date time.Time) string {
	return filepath.Join(b.baseDir, "nippo", date.Format("2006/01"))
}

func (b *FilesystemBackend) reportPath(date time.Time) string {
	return filepath.Join(b.reportDir(date), date.Format("02")+".md")
}

func (b *FilesystemBackend) Save(content string, date time.Time) error {
	dir := b.reportDir(date)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(b.reportPath(date), []byte(content), 0644)
}

func (b *FilesystemBackend) LoadReport(date time.Time) (string, error) {
	bytes, err := os.ReadFile(b.reportPath(date))
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

func (b *FilesystemBackend) LoadPreviousReport(date time.Time) (time.Time, error) {
	reports, err := b.ListReports()
	if err != nil {
		return time.Time{}, err
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
		return reportDate, nil
	}

	return time.Time{}, fs.ErrNotExist
}

func (b *FilesystemBackend) LoadLatestReport() (time.Time, error) {
	reports, err := b.ListReports()
	if err != nil {
		return time.Time{}, err
	}

	for _, rel := range reports {
		reportDate, err := parseReportDate(rel)
		if err != nil {
			continue
		}
		return reportDate, nil
	}

	return time.Time{}, fs.ErrNotExist
}

func (b *FilesystemBackend) ListReports() ([]string, error) {
	base := filepath.Join(b.baseDir, "nippo")
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

func (b *FilesystemBackend) Close() error {
	return nil
}
