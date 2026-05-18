package backends

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemBackend_SaveAndLoad(t *testing.T) {
	tests := []struct {
		name    string
		date    time.Time
		content string
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
		{
			name:    "multiline content",
			date:    time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			content: "# report\n\n## section\ncontent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			backend, err := NewFilesystemBackend(dir)
			require.NoError(t, err)

			err = backend.Save(tt.content, tt.date)
			require.NoError(t, err)

			loaded, err := backend.LoadReport(tt.date)
			require.NoError(t, err)
			assert.Equal(t, tt.content, loaded)
		})
	}
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
	tests := []struct {
		name          string
		saveDates     []time.Time
		expectedLen   int
		expectedFirst string
	}{
		{
			name: "multiple reports",
			saveDates: []time.Time{
				time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
				time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC),
				time.Date(2024, 5, 30, 0, 0, 0, 0, time.UTC),
			},
			expectedLen:   3,
			expectedFirst: "2024/06/15.md",
		},
		{
			name:          "empty",
			saveDates:     nil,
			expectedLen:   0,
			expectedFirst: "",
		},
		{
			name: "single report",
			saveDates: []time.Time{
				time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			},
			expectedLen:   1,
			expectedFirst: "2024/03/01.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			backend, err := NewFilesystemBackend(dir)
			require.NoError(t, err)

			for _, d := range tt.saveDates {
				err := backend.Save("# report", d)
				require.NoError(t, err)
			}

			reports, err := backend.ListReports()
			require.NoError(t, err)
			assert.Len(t, reports, tt.expectedLen)
			if tt.expectedLen > 0 {
				assert.Equal(t, tt.expectedFirst, reports[0])
			}
		})
	}
}

func TestFilesystemBackend_LoadPreviousReport(t *testing.T) {
	tests := []struct {
		name      string
		saveDates []string
		queryDate time.Time
		want      time.Time
		wantErr   bool
	}{
		{
			name:      "finds previous report",
			saveDates: []string{"2024-06-14", "2024-06-15"},
			queryDate: time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC),
			want:      time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "query date matches report date",
			saveDates: []string{"2024-06-14", "2024-06-15"},
			queryDate: time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			want:      time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "no previous report",
			saveDates: []string{"2024-06-15"},
			queryDate: time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC),
			wantErr:   true,
		},
		{
			name:      "empty storage",
			saveDates: []string{},
			queryDate: time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			backend, err := NewFilesystemBackend(dir)
			require.NoError(t, err)

			for _, dateStr := range tt.saveDates {
				d, _ := time.Parse("2006-01-02", dateStr)
				err := backend.Save("# report", d)
				require.NoError(t, err)
			}

			previous, err := backend.LoadPreviousReport(tt.queryDate)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, previous)
			}
		})
	}
}
