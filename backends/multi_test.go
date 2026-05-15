package backends

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiBackend_SaveAndLoad(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	fs1, err := NewFilesystemBackend(dir1)
	require.NoError(t, err)
	fs2, err := NewFilesystemBackend(dir2)
	require.NoError(t, err)

	multi := NewMultiBackend([]ReportStorage{fs1, fs2})

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	content := "# test report"

	err = multi.SaveReport(content, date)
	require.NoError(t, err)

	loaded1, err := fs1.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, content, loaded1)

	loaded2, err := fs2.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, content, loaded2)
}

func TestMultiBackend_LoadFromFirst(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	fs1, err := NewFilesystemBackend(dir1)
	require.NoError(t, err)
	fs2, err := NewFilesystemBackend(dir2)
	require.NoError(t, err)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	err = fs1.SaveReport("# from fs1", date)
	require.NoError(t, err)

	multi := NewMultiBackend([]ReportStorage{fs1, fs2})

	loaded, err := multi.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, "# from fs1", loaded)
}

func TestMultiBackend_ListReports(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	fs1, err := NewFilesystemBackend(dir1)
	require.NoError(t, err)
	fs2, err := NewFilesystemBackend(dir2)
	require.NoError(t, err)

	date1 := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC)

	err = fs1.SaveReport("# report 15", date1)
	require.NoError(t, err)
	err = fs2.SaveReport("# report 14", date2)
	require.NoError(t, err)

	multi := NewMultiBackend([]ReportStorage{fs1, fs2})

	reports, err := multi.ListReports()
	require.NoError(t, err)
	assert.Len(t, reports, 2)
}

func TestMultiBackend_LoadPreviousReport(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	fs1, err := NewFilesystemBackend(dir1)
	require.NoError(t, err)
	fs2, err := NewFilesystemBackend(dir2)
	require.NoError(t, err)

	date1 := time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	err = fs1.SaveReport("# report 14", date1)
	require.NoError(t, err)
	err = fs2.SaveReport("# report 15", date2)
	require.NoError(t, err)

	multi := NewMultiBackend([]ReportStorage{fs1, fs2})

	previous, err := multi.LoadPreviousReport(time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Contains(t, []string{"# report 14", "# report 15"}, previous)
}
