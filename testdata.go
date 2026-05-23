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
// 実際の generate と同じく .yaml と .md sidecar を両方書き、
// LoadPreviousReport (.md ベースの index) が前日日報を検出できる状態を作る。
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
