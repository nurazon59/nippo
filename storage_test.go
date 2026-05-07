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
	tmp := t.TempDir()
	storage, err := NewStorage(tmp)
	require.NoError(t, err)

	content := "# test"
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	err = storage.SaveReport(content, date)
	require.NoError(t, err)

	path := filepath.Join(tmp, "nippo", "2024", "06", "15.md")
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(got))
}

func TestLoadReport(t *testing.T) {
	tmp := t.TempDir()
	storage, err := NewStorage(tmp)
	require.NoError(t, err)

	content := "# test report"
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	err = storage.SaveReport(content, date)
	require.NoError(t, err)

	got, err := storage.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestLoadReportNotFound(t *testing.T) {
	tmp := t.TempDir()
	storage, err := NewStorage(tmp)
	require.NoError(t, err)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	_, err = storage.LoadReport(date)
	require.Error(t, err)
}

func TestListReports(t *testing.T) {
	tmp := t.TempDir()
	storage, err := NewStorage(tmp)
	require.NoError(t, err)

	dates := []time.Time{
		time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 5, 30, 0, 0, 0, 0, time.UTC),
	}

	for _, d := range dates {
		err := storage.SaveReport("# report", d)
		require.NoError(t, err)
	}

	reports, err := storage.ListReports()
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("2024", "06", "15.md"),
		filepath.Join("2024", "06", "14.md"),
		filepath.Join("2024", "05", "30.md"),
	}, reports)
}

func TestListReportsEmpty(t *testing.T) {
	tmp := t.TempDir()
	storage, err := NewStorage(tmp)
	require.NoError(t, err)

	reports, err := storage.ListReports()
	require.NoError(t, err)
	assert.Nil(t, reports)
}
