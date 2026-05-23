package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nurazon59/nippo/report"
)

func TestSaveReport(t *testing.T) {
	f := New(t)
	storage := f.NewStorage()

	content := "# test"
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	err := storage.SaveReport(content, date)
	require.NoError(t, err)

	path := filepath.Join(f.TmpDir(), "nippo", "2024", "06", "15.md")
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(got))
}

func TestLoadReport(t *testing.T) {
	f := New(t)
	f.SaveReport("2024-06-15", "# test report")
	storage := f.NewStorage()

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	got, err := storage.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, "# test report", got)
}

func TestLoadReportNotFound(t *testing.T) {
	f := New(t)
	storage := f.NewStorage()

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	_, err := storage.LoadReport(date)
	require.Error(t, err)
}

func TestListReports(t *testing.T) {
	f := New(t)
	f.SaveReport("2024-06-15", "# report")
	f.SaveReport("2024-06-14", "# report")
	f.SaveReport("2024-05-30", "# report")
	storage := f.NewStorage()

	reports, err := storage.ListReports()
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("2024", "06", "15.md"),
		filepath.Join("2024", "06", "14.md"),
		filepath.Join("2024", "05", "30.md"),
	}, reports)
}

func TestListReportsEmpty(t *testing.T) {
	f := New(t)
	storage := f.NewStorage()

	reports, err := storage.ListReports()
	require.NoError(t, err)
	assert.Nil(t, reports)
}

// TestSaveReportStructWritesCanonicalYAML は SaveReportStruct + WriteSidecar の組合せが
// .yaml と .md を同階層に並べることを e2e で担保する (Step 5 の主要な振る舞い)。
func TestSaveReportStructWritesCanonicalYAML(t *testing.T) {
	questions := []QuestionConfig{
		{Key: "done", Label: "やった"},
		{Key: "todo", Label: "やる"},
	}

	tests := map[string]struct {
		fields   map[string]string
		wantMD   string
		wantYAML string
	}{
		"text fields only": {
			fields: map[string]string{
				"done": "Aした",
				"todo": "Bする",
			},
			wantMD: "# 日報 2024-06-15\n\n## やった\nAした\n## やる\nBする\n",
			wantYAML: "schema_version: 1\n" +
				"date: \"2024-06-15\"\n" +
				"fields:\n" +
				"  done:\n" +
				"    type: text\n" +
				"    body: Aした\n" +
				"  todo:\n" +
				"    type: text\n" +
				"    body: Bする\n",
		},
		"empty field is preserved as empty text": {
			fields: map[string]string{
				"done": "",
				"todo": "",
			},
			wantMD: "# 日報 2024-06-15\n\n## やった\n\n## やる\n\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := New(t)
			storage := f.NewStorage()
			date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

			r := newTextReport(date, tt.fields)
			require.NoError(t, storage.SaveReportStruct(r))

			md, err := storage.LoadReportMarkdown(date, questions)
			require.NoError(t, err)
			require.NoError(t, storage.WriteSidecar(date, ".md", []byte(md)))

			yamlPath := filepath.Join(f.TmpDir(), "nippo", "2024", "06", "15.yaml")
			gotYAML, err := os.ReadFile(yamlPath)
			require.NoError(t, err)
			if tt.wantYAML != "" {
				assert.Equal(t, tt.wantYAML, string(gotYAML))
			}

			mdPath := filepath.Join(f.TmpDir(), "nippo", "2024", "06", "15.md")
			gotMD, err := os.ReadFile(mdPath)
			require.NoError(t, err)
			assert.Equal(t, tt.wantMD, string(gotMD))
			assert.Equal(t, tt.wantMD, md)
		})
	}
}

// TestLoadReportMarkdownFallback は .yaml 不在時に legacy .md が fallback されることを担保する。
// 移行期に .md だけ存在する過去日報を読めなくしない要件。
func TestLoadReportMarkdownFallback(t *testing.T) {
	questions := []QuestionConfig{
		{Key: "done", Label: "やった"},
	}

	tests := map[string]struct {
		setup    func(f *Fixture, storage *Storage, date time.Time)
		wantBody string
	}{
		"yaml present uses renderer": {
			setup: func(f *Fixture, storage *Storage, date time.Time) {
				require.NoError(t, storage.SaveReportStruct(newTextReport(date, map[string]string{"done": "X"})))
			},
			wantBody: "# 日報 2024-06-15\n\n## やった\nX\n",
		},
		"yaml absent falls back to legacy md": {
			setup: func(f *Fixture, storage *Storage, date time.Time) {
				require.NoError(t, storage.SaveReport("# legacy md\n", date))
			},
			wantBody: "# legacy md\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := New(t)
			storage := f.NewStorage()
			date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

			tt.setup(f, storage, date)

			got, err := storage.LoadReportMarkdown(date, questions)
			require.NoError(t, err)
			assert.Equal(t, tt.wantBody, got)
		})
	}
}

// TestLoadReportStructRoundTrip は SaveReportStruct → LoadReportStruct で同値に戻ることを担保する。
func TestLoadReportStructRoundTrip(t *testing.T) {
	f := New(t)
	storage := f.NewStorage()
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	want := newTextReport(date, map[string]string{"done": "やった", "todo": ""})
	require.NoError(t, storage.SaveReportStruct(want))

	got, err := storage.LoadReportStruct(date)
	require.NoError(t, err)
	assert.Equal(t, report.SupportedSchemaVersion, got.SchemaVersion)
	assert.Equal(t, want.Date, got.Date)
	assert.Equal(t, want.Fields, got.Fields)
}
