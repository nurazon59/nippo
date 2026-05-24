package backends

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/nurazon59/nippo/report"
)

type ReportStorage interface {
	Save(content string, date time.Time) error
	LoadReport(date time.Time) (string, error)
	LoadPreviousReport(date time.Time) (time.Time, error)
	LoadLatestReport() (time.Time, error)
	ListReports() ([]string, error)
	Close() error

	SaveReport(r *report.Report) error
	LoadReportStruct(date time.Time) (*report.Report, error)
	WriteSidecar(date time.Time, kind string, content []byte) error
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

func (b *FilesystemBackend) yamlReportPath(date time.Time) string {
	return filepath.Join(b.reportDir(date), date.Format("02")+".yaml")
}

func (b *FilesystemBackend) sidecarPath(date time.Time, kind string) string {
	return filepath.Join(b.reportDir(date), date.Format("02")+kind)
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

func (b *FilesystemBackend) SaveReport(r *report.Report) error {
	if r == nil {
		return fmt.Errorf("filesystem backend: report is nil")
	}
	buf, err := yaml.Marshal(r)
	if err != nil {
		return fmt.Errorf("filesystem backend: marshal yaml: %w", err)
	}
	dir := b.reportDir(r.Date)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(b.yamlReportPath(r.Date), buf, 0644)
}

func (b *FilesystemBackend) LoadReportStruct(date time.Time) (*report.Report, error) {
	bytes, err := os.ReadFile(b.yamlReportPath(date))
	if err != nil {
		return nil, err
	}
	var r report.Report
	if err := yaml.Unmarshal(bytes, &r); err != nil {
		return nil, fmt.Errorf("filesystem backend: unmarshal yaml: %w", err)
	}
	return &r, nil
}

func (b *FilesystemBackend) WriteSidecar(date time.Time, kind string, content []byte) error {
	if kind == "" {
		return fmt.Errorf("filesystem backend: sidecar kind is required")
	}
	dir := b.reportDir(date)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(b.sidecarPath(date, kind), content, 0644)
}
