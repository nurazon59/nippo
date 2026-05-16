package backends

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteBackend_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	backend, err := NewSQLiteBackend(path)
	require.NoError(t, err)
	defer backend.Close()

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	content := "# test report"

	err = backend.Save(content, date)
	require.NoError(t, err)

	loaded, err := backend.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, content, loaded)
}

func TestSQLiteBackend_LoadNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	backend, err := NewSQLiteBackend(path)
	require.NoError(t, err)
	defer backend.Close()

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	_, err = backend.LoadReport(date)
	require.Error(t, err)
}

func TestSQLiteBackend_ListReports(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	backend, err := NewSQLiteBackend(path)
	require.NoError(t, err)
	defer backend.Close()

	dates := []time.Time{
		time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 5, 30, 0, 0, 0, 0, time.UTC),
	}

	for _, d := range dates {
		err := backend.Save("# report", d)
		require.NoError(t, err)
	}

	reports, err := backend.ListReports()
	require.NoError(t, err)
	assert.Len(t, reports, 3)
	assert.Equal(t, "2024/06/15.md", reports[0])
}

func TestSQLiteBackend_LoadPreviousReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	backend, err := NewSQLiteBackend(path)
	require.NoError(t, err)
	defer backend.Close()

	date1 := time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	err = backend.Save("# report 14", date1)
	require.NoError(t, err)
	err = backend.Save("# report 15", date2)
	require.NoError(t, err)

	previous, err := backend.LoadPreviousReport(time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, date2, previous)
}

func TestSQLiteBackend_UpdateReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	backend, err := NewSQLiteBackend(path)
	require.NoError(t, err)
	defer backend.Close()

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	err = backend.Save("# original", date)
	require.NoError(t, err)

	err = backend.Save("# updated", date)
	require.NoError(t, err)

	loaded, err := backend.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, "# updated", loaded)
}

func TestSQLiteBackend_EmptyList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	backend, err := NewSQLiteBackend(path)
	require.NoError(t, err)
	defer backend.Close()

	reports, err := backend.ListReports()
	require.NoError(t, err)
	assert.Nil(t, reports)
}

func TestSQLiteBackend_NoPreviousReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	backend, err := NewSQLiteBackend(path)
	require.NoError(t, err)
	defer backend.Close()

	_, err = backend.LoadPreviousReport(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
	require.Error(t, err)
}
