package main

import (
	"errors"
	"io/fs"
	"time"

	"github.com/adrg/xdg"
	"github.com/nurazon59/nippo/backends"
	"github.com/nurazon59/nippo/renderer"
	"github.com/nurazon59/nippo/report"
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

// SaveReport は legacy の .md 直接保存。Step 5 では reference / edit / 既存テストの互換のため残す。
// 新規 cmd 層からは SaveReportStruct + WriteSidecar の組を使う。
func (s *Storage) SaveReport(content string, date time.Time) error {
	return s.backend.Save(content, date)
}

// LoadReport は legacy の .md 読み出し。reference.go と edit.go が利用中。
func (s *Storage) LoadReport(date time.Time) (string, error) {
	return s.backend.LoadReport(date)
}

// SaveReportStruct は canonical YAML を永続化する v1 保存口。
func (s *Storage) SaveReportStruct(r *report.Report) error {
	return s.backend.SaveReport(r)
}

// LoadReportStruct は canonical YAML を *report.Report として読み出す v1 読み出し口。
func (s *Storage) LoadReportStruct(date time.Time) (*report.Report, error) {
	return s.backend.LoadReportStruct(date)
}

// WriteSidecar は .md など canonical 派生物を保存する。
func (s *Storage) WriteSidecar(date time.Time, kind string, content []byte) error {
	return s.backend.WriteSidecar(date, kind, content)
}

// LoadReportMarkdown は .yaml ベースで render した markdown を返す。
// .yaml が無い場合は legacy .md を fallback として読み出し、移行期の互換性を保つ。
func (s *Storage) LoadReportMarkdown(date time.Time, questions []QuestionConfig) (string, error) {
	r, err := s.backend.LoadReportStruct(date)
	if err == nil {
		return renderer.Markdown(r, questionsToRendererQuestions(questions)), nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	return s.backend.LoadReport(date)
}

func (s *Storage) LoadPreviousReport(date time.Time) (time.Time, error) {
	return s.backend.LoadPreviousReport(date)
}

func (s *Storage) LoadLatestReport() (time.Time, error) {
	return s.backend.LoadLatestReport()
}

func (s *Storage) ListReports() ([]string, error) {
	return s.backend.ListReports()
}

func (s *Storage) Close() error {
	return s.backend.Close()
}
