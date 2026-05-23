package backends

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nurazon59/nippo/report"
)

func newTestSQLiteBackend(t *testing.T) *SQLiteBackend {
	t.Helper()
	path := filepath.Join(t.TempDir(), "reports.db")
	b, err := NewSQLiteBackend(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Close() })
	return b
}

type sqliteSave struct {
	date    string
	content string
}

func TestSQLiteBackend_SaveLoad(t *testing.T) {
	tests := map[string]struct {
		saves   []sqliteSave
		loadAt  string
		wantErr bool
		wantIs  error
		want    string
	}{
		"single insert": {
			saves:  []sqliteSave{{"2024-06-15", "# hello"}},
			loadAt: "2024-06-15",
			want:   "# hello",
		},
		"upsert keeps latest": {
			saves: []sqliteSave{
				{"2024-06-15", "first"},
				{"2024-06-15", "second"},
			},
			loadAt: "2024-06-15",
			want:   "second",
		},
		"miss": {
			saves:   nil,
			loadAt:  "2024-06-15",
			wantErr: true,
			wantIs:  fs.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b := newTestSQLiteBackend(t)
			for _, s := range tt.saves {
				require.NoError(t, b.Save(s.content, mustDate(t, s.date)))
			}

			got, err := b.LoadReport(mustDate(t, tt.loadAt))
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantIs != nil {
					assert.True(t, errors.Is(err, tt.wantIs))
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSQLiteBackend_LoadPreviousReport(t *testing.T) {
	tests := map[string]struct {
		saves   []string
		target  string
		wantErr bool
		wantIs  error
		want    string
	}{
		"picks newest before target": {
			saves:  []string{"2024-06-10", "2024-06-14"},
			target: "2024-06-15",
			want:   "2024-06-14",
		},
		"none before target": {
			saves:   nil,
			target:  "2024-06-15",
			wantErr: true,
			wantIs:  fs.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b := newTestSQLiteBackend(t)
			for _, d := range tt.saves {
				require.NoError(t, b.Save("x", mustDate(t, d)))
			}

			got, err := b.LoadPreviousReport(mustDate(t, tt.target))
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantIs != nil {
					assert.True(t, errors.Is(err, tt.wantIs))
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, mustDate(t, tt.want), got)
		})
	}
}

func TestSQLiteBackend_LoadLatestReport(t *testing.T) {
	tests := map[string]struct {
		saves   []string
		wantErr bool
		wantIs  error
		want    string
	}{
		"picks newest": {
			saves: []string{"2024-06-10", "2024-06-14", "2024-05-30"},
			want:  "2024-06-14",
		},
		"single entry": {
			saves: []string{"2024-06-15"},
			want:  "2024-06-15",
		},
		"empty": {
			saves:   nil,
			wantErr: true,
			wantIs:  fs.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b := newTestSQLiteBackend(t)
			for _, d := range tt.saves {
				require.NoError(t, b.Save("x", mustDate(t, d)))
			}

			got, err := b.LoadLatestReport()
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantIs != nil {
					assert.True(t, errors.Is(err, tt.wantIs))
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, mustDate(t, tt.want), got)
		})
	}
}

func TestSQLiteBackend_ListReports(t *testing.T) {
	tests := map[string]struct {
		saves []string
		want  []string
	}{
		"sorted desc": {
			saves: []string{"2024-06-14", "2024-06-15", "2024-05-30"},
			want: []string{
				filepath.Join("2024", "06", "15.md"),
				filepath.Join("2024", "06", "14.md"),
				filepath.Join("2024", "05", "30.md"),
			},
		},
		"empty": {
			saves: nil,
			want:  nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b := newTestSQLiteBackend(t)
			for _, d := range tt.saves {
				require.NoError(t, b.Save("x", mustDate(t, d)))
			}

			got, err := b.ListReports()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSQLiteBackend_SaveReportRoundTrip(t *testing.T) {
	tests := map[string]struct {
		date    string
		mutated bool
	}{
		"single insert":          {date: "2024-06-15"},
		"upsert keeps latest v1": {date: "2024-06-15", mutated: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b := newTestSQLiteBackend(t)
			r := sampleReport(t, tt.date)
			require.NoError(t, b.SaveReport(r))

			if tt.mutated {
				r2 := sampleReport(t, tt.date)
				r2.Fields["summary"] = report.FieldValue{Type: report.FieldTypeText, Body: "上書き"}
				require.NoError(t, b.SaveReport(r2))
				got, err := b.LoadReportStruct(mustDate(t, tt.date))
				require.NoError(t, err)
				assert.Equal(t, "上書き", got.Fields["summary"].Body)
				return
			}

			got, err := b.LoadReportStruct(mustDate(t, tt.date))
			require.NoError(t, err)
			assert.Equal(t, r.SchemaVersion, got.SchemaVersion)
			assert.True(t, r.Date.Equal(got.Date))
			assert.Equal(t, r.Fields, got.Fields)
		})
	}
}

func TestSQLiteBackend_LoadReportStructMissing(t *testing.T) {
	b := newTestSQLiteBackend(t)
	_, err := b.LoadReportStruct(mustDate(t, "2024-06-15"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestSQLiteBackend_WriteSidecarIsNoop(t *testing.T) {
	b := newTestSQLiteBackend(t)
	require.NoError(t, b.WriteSidecar(mustDate(t, "2024-06-15"), ".md", []byte("# x")))
}

func TestSQLiteBackend_LegacyAndV1Coexist(t *testing.T) {
	b := newTestSQLiteBackend(t)
	date := mustDate(t, "2024-06-15")

	require.NoError(t, b.Save("# legacy", date))
	require.NoError(t, b.SaveReport(sampleReport(t, "2024-06-15")))

	gotMD, err := b.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, "# legacy", gotMD)

	gotR, err := b.LoadReportStruct(date)
	require.NoError(t, err)
	assert.Equal(t, report.SupportedSchemaVersion, gotR.SchemaVersion)
}

func TestSQLiteBackend_NewValidation(t *testing.T) {
	tests := map[string]struct {
		path    func(t *testing.T) string
		wantErr bool
	}{
		"empty path errors": {
			path:    func(t *testing.T) string { return "" },
			wantErr: true,
		},
		"valid path ok": {
			path:    func(t *testing.T) string { return filepath.Join(t.TempDir(), "ok.db") },
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b, err := NewSQLiteBackend(tt.path(t))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			defer b.Close()
		})
	}
}
