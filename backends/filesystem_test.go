package backends

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemBackend_Save(t *testing.T) {
	dir := t.TempDir()
	b := NewFilesystemBackend(dir)

	content := "# test"
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	require.NoError(t, b.Save(content, date))

	path := filepath.Join(dir, "nippo", "2024", "06", "15.md")
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(got))
}

func TestFilesystemBackend_LoadReport(t *testing.T) {
	dir := t.TempDir()
	b := NewFilesystemBackend(dir)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	require.NoError(t, b.Save("# test report", date))

	got, err := b.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, "# test report", got)
}

func TestFilesystemBackend_LoadReportNotFound(t *testing.T) {
	dir := t.TempDir()
	b := NewFilesystemBackend(dir)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	_, err := b.LoadReport(date)
	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestFilesystemBackend_ListReports(t *testing.T) {
	dir := t.TempDir()
	b := NewFilesystemBackend(dir)

	for _, d := range []string{"2024-06-15", "2024-06-14", "2024-05-30"} {
		date, err := time.Parse("2006-01-02", d)
		require.NoError(t, err)
		require.NoError(t, b.Save("# report", date))
	}

	reports, err := b.ListReports()
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("2024", "06", "15.md"),
		filepath.Join("2024", "06", "14.md"),
		filepath.Join("2024", "05", "30.md"),
	}, reports)
}

func TestFilesystemBackend_ListReportsEmpty(t *testing.T) {
	dir := t.TempDir()
	b := NewFilesystemBackend(dir)

	reports, err := b.ListReports()
	require.NoError(t, err)
	assert.Nil(t, reports)
}

func TestFilesystemBackend_LoadPreviousReport(t *testing.T) {
	dir := t.TempDir()
	b := NewFilesystemBackend(dir)

	for _, d := range []string{"2024-06-10", "2024-06-14"} {
		date, err := time.Parse("2006-01-02", d)
		require.NoError(t, err)
		require.NoError(t, b.Save("# report", date))
	}

	target := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	got, err := b.LoadPreviousReport(target)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC), got)
}

func TestFilesystemBackend_LoadPreviousReportNotFound(t *testing.T) {
	dir := t.TempDir()
	b := NewFilesystemBackend(dir)

	target := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	_, err := b.LoadPreviousReport(target)
	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestFilesystemBackend_Close(t *testing.T) {
	b := NewFilesystemBackend(t.TempDir())
	require.NoError(t, b.Close())
}
