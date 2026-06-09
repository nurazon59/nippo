package main

import (
	"testing"
	"time"

	"github.com/nurazon59/nippo/report"
	"github.com/stretchr/testify/require"
)

type Fixture struct {
	t       *testing.T
	storage *Storage
	tmpDir  string
}

func New(t *testing.T) *Fixture {
	return &Fixture{t: t, tmpDir: t.TempDir()}
}

func (f *Fixture) NewStorage() *Storage {
	if f.storage == nil {
		s, err := NewStorage(&Config{StorageDir: f.tmpDir})
		require.NoError(f.t, err)
		f.storage = s
	}
	return f.storage
}

func (f *Fixture) TmpDir() string {
	return f.tmpDir
}

func (f *Fixture) SaveReport(date string, content string) {
	t, err := time.Parse("2006-01-02", date)
	require.NoError(f.t, err)
	require.NoError(f.t, f.NewStorage().SaveReport(content, t))
}

// SaveReportStruct は構造化 Report を保存するテスト用 helper。
// 構造化フローへの移行に伴い、buildReferencePresets が LoadReportStruct (.yaml) を
// 参照するため、テスト fixture も .yaml と .md sidecar を両方書く。
func (f *Fixture) SaveReportStruct(date string, fields map[string]report.FieldValue) {
	t, err := time.Parse("2006-01-02", date)
	require.NoError(f.t, err)
	r := &report.Report{
		SchemaVersion: report.SupportedSchemaVersion,
		Date:          t,
		Fields:        fields,
	}
	s := f.NewStorage()
	require.NoError(f.t, s.SaveReportStruct(r))
	require.NoError(f.t, s.WriteSidecar(t, ".md", []byte("# 日報 "+date+"\n")))
}

func (f *Fixture) LoadConfig(path string) *Config {
	cfg, err := Load(path)
	require.NoError(f.t, err)
	return cfg
}

func (f *Fixture) DefaultConfig() *Config {
	return Default()
}

// newTextReport は「全フィールド text 型」のテスト用 Report を組み立てる test helper。
// 本番コードは runForm が直接 Report を構築するため、テスト便宜のためにのみ存在する。
func newTextReport(date time.Time, fields map[string]string) *report.Report {
	r := &report.Report{
		SchemaVersion: report.SupportedSchemaVersion,
		Date:          date,
		Fields:        make(map[string]report.FieldValue, len(fields)),
	}
	for k, v := range fields {
		r.Fields[k] = report.FieldValue{Type: report.FieldTypeText, Body: v}
	}
	return r
}
