package backends

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSQLiteBackend(t *testing.T) *SQLiteBackend {
	t.Helper()
	path := filepath.Join(t.TempDir(), "reports.db")
	b, err := NewSQLiteBackend(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Close() })
	return b
}

func TestSQLiteBackend_SaveLoad(t *testing.T) {
	b := newTestSQLiteBackend(t)
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	require.NoError(t, b.Save("# hello", date))

	got, err := b.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, "# hello", got)
}

func TestSQLiteBackend_SaveUpsert(t *testing.T) {
	b := newTestSQLiteBackend(t)
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	require.NoError(t, b.Save("first", date))
	require.NoError(t, b.Save("second", date))

	got, err := b.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, "second", got)
}

func TestSQLiteBackend_LoadNotFound(t *testing.T) {
	b := newTestSQLiteBackend(t)
	_, err := b.LoadReport(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestSQLiteBackend_LoadPreviousReport(t *testing.T) {
	b := newTestSQLiteBackend(t)
	require.NoError(t, b.Save("old", time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)))
	require.NoError(t, b.Save("newer", time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC)))

	got, err := b.LoadPreviousReport(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC), got)
}

func TestSQLiteBackend_LoadPreviousReportNotFound(t *testing.T) {
	b := newTestSQLiteBackend(t)
	_, err := b.LoadPreviousReport(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestSQLiteBackend_ListReports(t *testing.T) {
	b := newTestSQLiteBackend(t)
	for _, d := range []string{"2024-06-14", "2024-06-15", "2024-05-30"} {
		date, err := time.Parse("2006-01-02", d)
		require.NoError(t, err)
		require.NoError(t, b.Save("x", date))
	}

	got, err := b.ListReports()
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("2024", "06", "15.md"),
		filepath.Join("2024", "06", "14.md"),
		filepath.Join("2024", "05", "30.md"),
	}, got)
}

func TestSQLiteBackend_NewRequiresPath(t *testing.T) {
	_, err := NewSQLiteBackend("")
	require.Error(t, err)
}
