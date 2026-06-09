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

	"github.com/nurazon59/nippo/report"
)

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	require.NoError(t, err)
	return d
}

func TestFilesystemBackend_Save(t *testing.T) {
	tests := map[string]struct {
		date    string
		content string
		wantRel string
	}{
		"basic":     {date: "2024-06-15", content: "# test", wantRel: filepath.Join("2024", "06", "15.md")},
		"different": {date: "2024-12-01", content: "x", wantRel: filepath.Join("2024", "12", "01.md")},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			b := NewFilesystemBackend(dir)

			require.NoError(t, b.Save(tt.content, mustDate(t, tt.date)))

			got, err := os.ReadFile(filepath.Join(dir, "nippo", tt.wantRel))
			require.NoError(t, err)
			assert.Equal(t, tt.content, string(got))
		})
	}
}

func TestFilesystemBackend_LoadReport(t *testing.T) {
	tests := map[string]struct {
		setup   func(b *FilesystemBackend, t *testing.T)
		date    string
		wantErr bool
		wantIs  error
		want    string
	}{
		"hit": {
			setup: func(b *FilesystemBackend, t *testing.T) {
				require.NoError(t, b.Save("# test report", mustDate(t, "2024-06-15")))
			},
			date: "2024-06-15",
			want: "# test report",
		},
		"miss": {
			setup:   func(b *FilesystemBackend, t *testing.T) {},
			date:    "2024-06-15",
			wantErr: true,
			wantIs:  fs.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b := NewFilesystemBackend(t.TempDir())
			tt.setup(b, t)

			got, err := b.LoadReport(mustDate(t, tt.date))
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

func TestFilesystemBackend_ListReports(t *testing.T) {
	tests := map[string]struct {
		saves []string
		want  []string
	}{
		"sorted desc": {
			saves: []string{"2024-06-15", "2024-06-14", "2024-05-30"},
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
			b := NewFilesystemBackend(t.TempDir())
			for _, d := range tt.saves {
				require.NoError(t, b.Save("# r", mustDate(t, d)))
			}

			got, err := b.ListReports()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilesystemBackend_LoadPreviousReport(t *testing.T) {
	tests := map[string]struct {
		saves   []string
		target  string
		wantErr bool
		wantIs  error
		want    string
	}{
		"picks immediate previous": {
			saves:  []string{"2024-06-10", "2024-06-14"},
			target: "2024-06-15",
			want:   "2024-06-14",
		},
		"no history": {
			saves:   nil,
			target:  "2024-06-15",
			wantErr: true,
			wantIs:  fs.ErrNotExist,
		},
		"same day excluded": {
			saves:   []string{"2024-06-15"},
			target:  "2024-06-15",
			wantErr: true,
			wantIs:  fs.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b := NewFilesystemBackend(t.TempDir())
			for _, d := range tt.saves {
				require.NoError(t, b.Save("# r", mustDate(t, d)))
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

func TestFilesystemBackend_LoadLatestReport(t *testing.T) {
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
			b := NewFilesystemBackend(t.TempDir())
			for _, d := range tt.saves {
				require.NoError(t, b.Save("# r", mustDate(t, d)))
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

func TestFilesystemBackend_Close(t *testing.T) {
	b := NewFilesystemBackend(t.TempDir())
	require.NoError(t, b.Close())
}

func sampleReport(t *testing.T, date string) *report.Report {
	t.Helper()
	return &report.Report{
		SchemaVersion: report.SupportedSchemaVersion,
		Date:          mustDate(t, date),
		Fields: map[string]report.FieldValue{
			"summary": {Type: report.FieldTypeText, Body: "今日の振り返り"},
			"tasks": {
				Type: report.FieldTypeTaskList,
				Tasks: []report.Task{
					{Title: "実装", Time: "2h", Outcome: "完了"},
				},
			},
		},
	}
}

func TestFilesystemBackend_SaveReportRoundTrip(t *testing.T) {
	tests := map[string]struct {
		date    string
		wantRel string
	}{
		"basic":    {date: "2024-06-15", wantRel: filepath.Join("2024", "06", "15.yaml")},
		"year end": {date: "2024-12-31", wantRel: filepath.Join("2024", "12", "31.yaml")},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			b := NewFilesystemBackend(dir)
			r := sampleReport(t, tt.date)

			require.NoError(t, b.SaveReport(r))

			_, err := os.Stat(filepath.Join(dir, "nippo", tt.wantRel))
			require.NoError(t, err)

			got, err := b.LoadReportStruct(mustDate(t, tt.date))
			require.NoError(t, err)
			assert.Equal(t, r.SchemaVersion, got.SchemaVersion)
			assert.True(t, r.Date.Equal(got.Date), "date mismatch: want=%v got=%v", r.Date, got.Date)
			assert.Equal(t, r.Fields, got.Fields)
		})
	}
}

func TestFilesystemBackend_SaveReportRejectsNil(t *testing.T) {
	b := NewFilesystemBackend(t.TempDir())
	require.Error(t, b.SaveReport(nil))
}

func TestFilesystemBackend_LoadReportStructMissing(t *testing.T) {
	b := NewFilesystemBackend(t.TempDir())
	_, err := b.LoadReportStruct(mustDate(t, "2024-06-15"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestFilesystemBackend_WriteSidecar(t *testing.T) {
	tests := map[string]struct {
		date    string
		kind    string
		content string
		wantRel string
		wantErr bool
	}{
		"markdown sidecar": {
			date:    "2024-06-15",
			kind:    ".md",
			content: "# nippo",
			wantRel: filepath.Join("2024", "06", "15.md"),
		},
		"html sidecar": {
			date:    "2024-07-01",
			kind:    ".html",
			content: "<h1>x</h1>",
			wantRel: filepath.Join("2024", "07", "01.html"),
		},
		"empty kind errors": {
			date:    "2024-06-15",
			kind:    "",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			b := NewFilesystemBackend(dir)

			err := b.WriteSidecar(mustDate(t, tt.date), tt.kind, []byte(tt.content))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			got, err := os.ReadFile(filepath.Join(dir, "nippo", tt.wantRel))
			require.NoError(t, err)
			assert.Equal(t, tt.content, string(got))
		})
	}
}

func TestFilesystemBackend_LegacyAndV1Coexist(t *testing.T) {
	dir := t.TempDir()
	b := NewFilesystemBackend(dir)
	date := mustDate(t, "2024-06-15")

	require.NoError(t, b.Save("# legacy md", date))
	require.NoError(t, b.SaveReport(sampleReport(t, "2024-06-15")))

	gotMD, err := b.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, "# legacy md", gotMD)

	gotR, err := b.LoadReportStruct(date)
	require.NoError(t, err)
	assert.Equal(t, report.SupportedSchemaVersion, gotR.SchemaVersion)
}

// Step 4: SaveReport (.yaml) と WriteSidecar (.md など) を同 date で並べたときの
// 配置・冪等性・kind 拡張性・親ディレクトリ自動作成をまとめて検証する。
// Step 3 で実装した sibling 配置の意図を破壊的に壊さないためのカバレッジ強化。
func TestFilesystemBackend_SidecarCoexistence(t *testing.T) {
	tests := map[string]struct {
		writes []struct {
			kind    string
			content string
		}
		date           string
		wantYAMLRel    string
		wantSidecarRel string
		wantSidecar    string
		wantListed     []string
	}{
		"md sidecar lives next to yaml and shows up in ListReports": {
			date: "2024-06-15",
			writes: []struct {
				kind    string
				content string
			}{
				{kind: ".md", content: "# rendered nippo"},
			},
			wantYAMLRel:    filepath.Join("2024", "06", "15.yaml"),
			wantSidecarRel: filepath.Join("2024", "06", "15.md"),
			wantSidecar:    "# rendered nippo",
			wantListed:     []string{filepath.Join("2024", "06", "15.md")},
		},
		"second write overwrites first (idempotent)": {
			date: "2024-06-15",
			writes: []struct {
				kind    string
				content string
			}{
				{kind: ".md", content: "# first"},
				{kind: ".md", content: "# second"},
			},
			wantYAMLRel:    filepath.Join("2024", "06", "15.yaml"),
			wantSidecarRel: filepath.Join("2024", "06", "15.md"),
			wantSidecar:    "# second",
			wantListed:     []string{filepath.Join("2024", "06", "15.md")},
		},
		"json sidecar (non-md kind) coexists without polluting ListReports": {
			date: "2024-07-01",
			writes: []struct {
				kind    string
				content string
			}{
				{kind: ".json", content: `{"ok":true}`},
			},
			wantYAMLRel:    filepath.Join("2024", "07", "01.yaml"),
			wantSidecarRel: filepath.Join("2024", "07", "01.json"),
			wantSidecar:    `{"ok":true}`,
			wantListed:     nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			b := NewFilesystemBackend(dir)
			date := mustDate(t, tt.date)

			require.NoError(t, b.SaveReport(sampleReport(t, tt.date)))

			for _, w := range tt.writes {
				require.NoError(t, b.WriteSidecar(date, w.kind, []byte(w.content)))
			}

			yamlAbs := filepath.Join(dir, "nippo", tt.wantYAMLRel)
			sidecarAbs := filepath.Join(dir, "nippo", tt.wantSidecarRel)
			assert.Equal(t, filepath.Dir(yamlAbs), filepath.Dir(sidecarAbs), "yaml と sidecar が別ディレクトリに分散している")

			_, err := os.Stat(yamlAbs)
			require.NoError(t, err, "yaml が想定パスに存在しない: %s", yamlAbs)

			gotSidecar, err := os.ReadFile(sidecarAbs)
			require.NoError(t, err, "sidecar が想定パスに存在しない: %s", sidecarAbs)
			assert.Equal(t, tt.wantSidecar, string(gotSidecar))

			gotR, err := b.LoadReportStruct(date)
			require.NoError(t, err)
			assert.Equal(t, report.SupportedSchemaVersion, gotR.SchemaVersion)

			gotList, err := b.ListReports()
			require.NoError(t, err)
			assert.Equal(t, tt.wantListed, gotList)
		})
	}
}
