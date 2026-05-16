package backends

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiBackend_SaveAndLoad(t *testing.T) {
	tests := []struct {
		name    string
		content string
		date    time.Time
	}{
		{
			name:    "basic report",
			date:    time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			content: "# test report",
		},
		{
			name:    "empty content",
			date:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			content: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir1 := t.TempDir()
			dir2 := t.TempDir()

			fs1, err := NewFilesystemBackend(dir1)
			require.NoError(t, err)
			fs2, err := NewFilesystemBackend(dir2)
			require.NoError(t, err)

			multi := NewMultiBackend([]ReportStorage{fs1, fs2})

			err = multi.Save(tt.content, tt.date)
			require.NoError(t, err)

			loaded1, err := fs1.LoadReport(tt.date)
			require.NoError(t, err)
			assert.Equal(t, tt.content, loaded1)

			loaded2, err := fs2.LoadReport(tt.date)
			require.NoError(t, err)
			assert.Equal(t, tt.content, loaded2)
		})
	}
}

func TestMultiBackend_LoadFromFirst(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	fs1, err := NewFilesystemBackend(dir1)
	require.NoError(t, err)
	fs2, err := NewFilesystemBackend(dir2)
	require.NoError(t, err)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	err = fs1.Save("# from fs1", date)
	require.NoError(t, err)

	multi := NewMultiBackend([]ReportStorage{fs1, fs2})

	loaded, err := multi.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, "# from fs1", loaded)
}

func TestMultiBackend_ListReports(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(fs1, fs2 *FilesystemBackend)
		expectedLen int
	}{
		{
			name: "reports in both backends",
			setup: func(fs1, fs2 *FilesystemBackend) {
				fs1.Save("# report", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
				fs2.Save("# report", time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC))
			},
			expectedLen: 2,
		},
		{
			name:        "empty backends",
			setup:       func(fs1, fs2 *FilesystemBackend) {},
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir1 := t.TempDir()
			dir2 := t.TempDir()

			fs1, err := NewFilesystemBackend(dir1)
			require.NoError(t, err)
			fs2, err := NewFilesystemBackend(dir2)
			require.NoError(t, err)

			tt.setup(fs1, fs2)

			multi := NewMultiBackend([]ReportStorage{fs1, fs2})

			reports, err := multi.ListReports()
			require.NoError(t, err)
			assert.Len(t, reports, tt.expectedLen)
		})
	}
}

func TestMultiBackend_LoadPreviousReport(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(fs1, fs2 *FilesystemBackend)
		queryDate time.Time
		want      time.Time
		wantErr   bool
	}{
		{
			name: "finds previous report",
			setup: func(fs1, fs2 *FilesystemBackend) {
				fs1.Save("# report 14", time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC))
				fs2.Save("# report 15", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
			},
			queryDate: time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC),
			want:      time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "no previous report",
			setup:     func(fs1, fs2 *FilesystemBackend) {},
			queryDate: time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir1 := t.TempDir()
			dir2 := t.TempDir()

			fs1, err := NewFilesystemBackend(dir1)
			require.NoError(t, err)
			fs2, err := NewFilesystemBackend(dir2)
			require.NoError(t, err)

			tt.setup(fs1, fs2)

			multi := NewMultiBackend([]ReportStorage{fs1, fs2})

			previous, err := multi.LoadPreviousReport(tt.queryDate)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, previous)
			}
		})
	}
}
