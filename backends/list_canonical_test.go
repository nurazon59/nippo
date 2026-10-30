package backends

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/nurazon59/nippo/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListReportsIncludesCanonicalYAML(t *testing.T) {
	tests := map[string]func(string) (ReportStorage, error){
		"filesystem": func(dir string) (ReportStorage, error) {
			return NewFilesystemBackend(dir), nil
		},
		"sqlite": func(dir string) (ReportStorage, error) {
			return NewSQLiteBackend(filepath.Join(dir, "reports.db"))
		},
	}
	for name, build := range tests {
		t.Run(name, func(t *testing.T) {
			storage, err := build(t.TempDir())
			require.NoError(t, err)
			defer storage.Close()

			date := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
			r := &report.Report{SchemaVersion: report.SupportedSchemaVersion, Date: date}
			require.NoError(t, storage.SaveReport(r))

			// YAML 正本だけの日報も一覧に含める。
			paths, err := storage.ListReports()
			require.NoError(t, err)
			assert.Equal(t, []string{filepath.Join("2026", "10", "30.md")}, paths)

			// legacy Markdown と同じ日付でも重複しない。
			require.NoError(t, storage.Save("legacy", date))
			paths, err = storage.ListReports()
			require.NoError(t, err)
			assert.Equal(t, []string{filepath.Join("2026", "10", "30.md")}, paths)
		})
	}
}
