package backends

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemBackend_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	backend, err := NewFilesystemBackend(dir)
	require.NoError(t, err)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	content := "# test report"

	err = backend.SaveReport(content, date)
	require.NoError(t, err)

	loaded, err := backend.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, content, loaded)
}

func TestFilesystemBackend_LoadNotFound(t *testing.T) {
	dir := t.TempDir()
	backend, err := NewFilesystemBackend(dir)
	require.NoError(t, err)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	_, err = backend.LoadReport(date)
	require.Error(t, err)
}

func TestFilesystemBackend_ListReports(t *testing.T) {
	dir := t.TempDir()
	backend, err := NewFilesystemBackend(dir)
	require.NoError(t, err)

	dates := []time.Time{
		time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 5, 30, 0, 0, 0, 0, time.UTC),
	}

	for _, d := range dates {
		err := backend.SaveReport("# report", d)
		require.NoError(t, err)
	}

	reports, err := backend.ListReports()
	require.NoError(t, err)
	assert.Len(t, reports, 3)
	assert.Equal(t, "2024/06/15.md", reports[0])
}

func TestFilesystemBackend_LoadPreviousReport(t *testing.T) {
	dir := t.TempDir()
	backend, err := NewFilesystemBackend(dir)
	require.NoError(t, err)

	date1 := time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	err = backend.SaveReport("# report 14", date1)
	require.NoError(t, err)
	err = backend.SaveReport("# report 15", date2)
	require.NoError(t, err)

	previous, err := backend.LoadPreviousReport(time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, "# report 15", previous)
}
