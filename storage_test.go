package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveReport(t *testing.T) {
	f := New(t)
	storage := f.NewStorage()

	content := "# test"
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	err := storage.SaveReport(content, date)
	require.NoError(t, err)

	path := filepath.Join(storage.baseDir, "nippo", "2024", "06", "15.md")
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(got))
}

func TestLoadReport(t *testing.T) {
	f := New(t)
	f.SaveReport("2024-06-15", "# test report")
	storage := f.NewStorage()

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	got, err := storage.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, "# test report", got)
}

func TestLoadReportNotFound(t *testing.T) {
	f := New(t)
	storage := f.NewStorage()

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	_, err := storage.LoadReport(date)
	require.Error(t, err)
}

func TestListReports(t *testing.T) {
	f := New(t)
	f.SaveReport("2024-06-15", "# report")
	f.SaveReport("2024-06-14", "# report")
	f.SaveReport("2024-05-30", "# report")
	storage := f.NewStorage()

	reports, err := storage.ListReports()
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("2024", "06", "15.md"),
		filepath.Join("2024", "06", "14.md"),
		filepath.Join("2024", "05", "30.md"),
	}, reports)
}

func TestListReportsEmpty(t *testing.T) {
	f := New(t)
	storage := f.NewStorage()

	reports, err := storage.ListReports()
	require.NoError(t, err)
	assert.Nil(t, reports)
}
