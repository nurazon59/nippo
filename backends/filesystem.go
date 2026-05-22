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

// ReportStorage は backend の抽象。
// Step 3 では新規 v1 メソッド (SaveReport / LoadReportStruct / WriteSidecar) を
// 既存の legacy メソッドに追加する形で導入し、Step 5 以降で wiring を切り替える。
type ReportStorage interface {
	// === 既存 (legacy: .md ベース) ===
	Save(content string, date time.Time) error
	LoadReport(date time.Time) (string, error)
	LoadPreviousReport(date time.Time) (time.Time, error)
	LoadLatestReport() (time.Time, error)
	ListReports() ([]string, error)
	Close() error

	// === 新規 (v1 structured: canonical YAML + sidecar) ===
	// SaveReport は canonical YAML を永続化する。
	SaveReport(r *report.Report) error
	// LoadReportStruct は canonical YAML を *report.Report に復元する。
	LoadReportStruct(date time.Time) (*report.Report, error)
	// WriteSidecar は同日に紐づく副生成物 (例: .md) を保存する。
	// kind は拡張子付き文字列 (e.g. ".md")。backend によっては no-op で良い。
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

// yamlReportPath は v1 canonical YAML の保存先。
// .md の隣に YYYY/MM/DD.yaml で配置し、Step 4 以降で git backend からも参照する。
func (b *FilesystemBackend) yamlReportPath(date time.Time) string {
	return filepath.Join(b.reportDir(date), date.Format("02")+".yaml")
}

// sidecarPath は kind (例: ".md") を suffix として DD<kind> に解決する。
// kind 先頭が "." でない場合も "." を補わずそのまま結合し、呼び出し側の意図を尊重する。
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

// SaveReport は canonical YAML を YYYY/MM/DD.yaml に書き出す。
// 既存 Save (.md) には影響しない (Step 5 で wiring を切り替えるまで共存)。
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

// LoadReportStruct は YYYY/MM/DD.yaml を読み出し *report.Report に復元する。
// 不在時は io/fs.ErrNotExist を errors.Is で検出できる形で返す。
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

// WriteSidecar は kind を suffix としてファイルを書き出す。
// 例: kind=".md" なら YYYY/MM/DD.md。canonical YAML と同じ階層に並べる。
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
