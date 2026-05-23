package main

import (
	"bytes"
	"encoding/json"
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

// TestParseLegacyMarkdown は `## ` heading の section 切り出しを担保する。
// CRLF・先頭の `# 日報` 行・空行多発などのケースで body が壊れないことを検証する。
func TestParseLegacyMarkdown(t *testing.T) {
	tests := map[string]struct {
		content string
		want    map[string]string
	}{
		"3 section text body": {
			content: "# 日報 2024-06-15\n\n## やった\nAをした\nBもした\n\n## やる\nCする\n\n## 所感\n順調\n",
			want: map[string]string{
				"やった": "Aをした\nBもした",
				"やる":  "Cする",
				"所感":  "順調",
			},
		},
		"crlf line endings are normalized": {
			content: "# 日報 2024-06-15\r\n\r\n## やった\r\nAをした\r\n\r\n## やる\r\nCする\r\n",
			want: map[string]string{
				"やった": "Aをした",
				"やる":  "Cする",
			},
		},
		"empty body section yields empty string": {
			content: "## やった\n\n## やる\nCする\n",
			want: map[string]string{
				"やった": "",
				"やる":  "Cする",
			},
		},
		"no section yields empty map": {
			content: "# 日報 2024-06-15\nthis has no sections\n",
			want:    map[string]string{},
		},
		"trailing whitespace on header trimmed": {
			content: "## やった   \nbody\n",
			want:    map[string]string{"やった": "body"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseLegacyMarkdown(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestParseLegacyTaskBullets は renderer.writeTask の逆変換を担保する。
// title/time/outcome/thoughts の各組合せが round-trip 可能であることを示す。
func TestParseLegacyTaskBullets(t *testing.T) {
	tests := map[string]struct {
		body string
		want []report.Task
	}{
		"title only": {
			body: "- A\n",
			want: []report.Task{{Title: "A"}},
		},
		"title with time": {
			body: "- A (30m)\n",
			want: []report.Task{{Title: "A", Time: "30m"}},
		},
		"title with time and outcome": {
			body: "- A (30m) done\n",
			want: []report.Task{{Title: "A", Time: "30m", Outcome: "done"}},
		},
		"title with thoughts": {
			body: "- A\n  順調だった\n",
			want: []report.Task{{Title: "A", Thoughts: "順調だった"}},
		},
		"full task": {
			body: "- A (1h) done\n  順調\n",
			want: []report.Task{{Title: "A", Time: "1h", Outcome: "done", Thoughts: "順調"}},
		},
		"multiple tasks": {
			body: "- A (30m) done\n  懸念点\n- B\n",
			want: []report.Task{
				{Title: "A", Time: "30m", Outcome: "done", Thoughts: "懸念点"},
				{Title: "B"},
			},
		},
		"empty body yields empty slice": {
			body: "",
			want: []report.Task{},
		},
		"free text without bullets is dropped": {
			body: "自由文だけ\n  インデント無し対象外",
			want: []report.Task{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseLegacyTaskBullets(tt.body)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestMigrateProcessOne は processOne の分岐を網羅する。
// .yaml 既存 / 0 section / 正常 / cfg.Questions 不一致 / apply 効果 (副作用) を 1 テーブルで担保する。
func TestMigrateProcessOne(t *testing.T) {
	defaultQuestions := []QuestionConfig{
		{Key: "done", Label: "やった"},
		{Key: "todo", Label: "やる"},
		{Key: "tasks", Label: "タスク", Type: "task_list"},
	}

	tests := map[string]struct {
		apply           bool
		setupFunc       func(f *Fixture, date time.Time)
		date            time.Time
		questions       []QuestionConfig
		wantAction      migrateAction
		wantSections    []string
		wantUnknown     []string
		wantReason      string
		wantYAMLWritten bool
		wantFields      map[string]report.FieldValue
	}{
		"existing yaml is skipped": {
			apply: true,
			setupFunc: func(f *Fixture, date time.Time) {
				require.NoError(f.t, f.NewStorage().SaveReport("# 日報 2024-06-15\n\n## やった\nx\n", date))
				require.NoError(f.t, f.NewStorage().SaveReportStruct(&report.Report{
					SchemaVersion: report.SupportedSchemaVersion,
					Date:          date,
					Fields:        map[string]report.FieldValue{"done": {Type: report.FieldTypeText, Body: "既存"}},
				}))
			},
			date:            time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			questions:       defaultQuestions,
			wantAction:      migrateActionSkip,
			wantReason:      "already has .yaml",
			wantYAMLWritten: true, // 既存 yaml は維持される
			wantFields: map[string]report.FieldValue{
				"done": {Type: report.FieldTypeText, Body: "既存"},
			},
		},
		"no sections is skipped": {
			apply: true,
			setupFunc: func(f *Fixture, date time.Time) {
				require.NoError(f.t, f.NewStorage().SaveReport("自由文だけ\n", date))
			},
			date:       time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			questions:  defaultQuestions,
			wantAction: migrateActionSkip,
			wantReason: "no recognizable sections",
		},
		"section が cfg.Questions に一致しない場合 skip + warn": {
			apply: true,
			setupFunc: func(f *Fixture, date time.Time) {
				require.NoError(f.t, f.NewStorage().SaveReport("## 別のヘッダ\nbody\n", date))
			},
			date:        time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			questions:   defaultQuestions,
			wantAction:  migrateActionSkip,
			wantUnknown: []string{"別のヘッダ"},
			wantReason:  "no sections matched configured questions",
		},
		"dry-run does not write yaml": {
			apply: false,
			setupFunc: func(f *Fixture, date time.Time) {
				require.NoError(f.t, f.NewStorage().SaveReport("## やった\nやったこと\n\n## やる\nやること\n", date))
			},
			date:            time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			questions:       defaultQuestions,
			wantAction:      migrateActionMigrate,
			wantSections:    []string{"done", "todo"},
			wantYAMLWritten: false,
		},
		"apply writes yaml with text fields": {
			apply: true,
			setupFunc: func(f *Fixture, date time.Time) {
				require.NoError(f.t, f.NewStorage().SaveReport("# 日報 2024-06-15\n\n## やった\nやったこと\n\n## やる\nやること\n", date))
			},
			date:            time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			questions:       defaultQuestions,
			wantAction:      migrateActionMigrate,
			wantSections:    []string{"done", "todo"},
			wantYAMLWritten: true,
			wantFields: map[string]report.FieldValue{
				"done": {Type: report.FieldTypeText, Body: "やったこと"},
				"todo": {Type: report.FieldTypeText, Body: "やること"},
			},
		},
		"apply parses task_list bullets": {
			apply: true,
			setupFunc: func(f *Fixture, date time.Time) {
				require.NoError(f.t, f.NewStorage().SaveReport(
					"## タスク\n- 設計レビュー (30m) done\n  懸念を整理\n- 実装\n", date))
			},
			date:            time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			questions:       defaultQuestions,
			wantAction:      migrateActionMigrate,
			wantSections:    []string{"tasks"},
			wantYAMLWritten: true,
			wantFields: map[string]report.FieldValue{
				"tasks": {Type: report.FieldTypeTaskList, Tasks: []report.Task{
					{Title: "設計レビュー", Time: "30m", Outcome: "done", Thoughts: "懸念を整理"},
					{Title: "実装"},
				}},
			},
		},
		"未知 heading は警告のみで他 section は migrate": {
			apply: true,
			setupFunc: func(f *Fixture, date time.Time) {
				require.NoError(f.t, f.NewStorage().SaveReport(
					"## やった\nx\n\n## 未知ヘッダ\nignored\n", date))
			},
			date:            time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			questions:       defaultQuestions,
			wantAction:      migrateActionMigrate,
			wantSections:    []string{"done"},
			wantUnknown:     []string{"未知ヘッダ"},
			wantYAMLWritten: true,
			wantFields: map[string]report.FieldValue{
				"done": {Type: report.FieldTypeText, Body: "x"},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := New(t)
			cfg := &Config{
				Version:    1,
				StorageDir: f.TmpDir(),
				Questions:  tt.questions,
			}
			tt.setupFunc(f, tt.date)

			c := &migrateCmd{Apply: tt.apply}
			storage, err := NewStorage(cfg)
			require.NoError(t, err)
			defer storage.Close()

			res := c.processOne(storage, cfg, tt.date)
			assert.Equal(t, tt.wantAction, res.Action, "action")
			assert.Equal(t, tt.wantSections, res.Sections, "sections")
			assert.Equal(t, tt.wantUnknown, res.UnknownSections, "unknown")
			if tt.wantReason != "" {
				assert.Equal(t, tt.wantReason, res.Reason, "reason")
			}

			yamlPath := filepath.Join(f.TmpDir(), "nippo",
				tt.date.Format("2006"), tt.date.Format("01"), tt.date.Format("02")+".yaml")
			_, statErr := os.Stat(yamlPath)
			if tt.wantYAMLWritten {
				require.NoError(t, statErr, "yaml must exist at %s", yamlPath)
				got, err := storage.LoadReportStruct(tt.date)
				require.NoError(t, err)
				if tt.wantFields != nil {
					assert.Equal(t, tt.wantFields, got.Fields)
				}
			} else {
				require.True(t, errors.Is(statErr, fs.ErrNotExist), "yaml must not be written, got stat err=%v", statErr)
			}
		})
	}
}

// TestMigrateFindTargets は --date / ListReports 経由の対象決定を担保する。
func TestMigrateFindTargets(t *testing.T) {
	tests := map[string]struct {
		date      string
		setupFunc func(f *Fixture)
		wantDates []string
		wantErr   bool
	}{
		"single date explicit": {
			date:      "2024-06-15",
			setupFunc: func(f *Fixture) {},
			wantDates: []string{"2024-06-15"},
		},
		"invalid date returns error": {
			date:      "invalid",
			setupFunc: func(f *Fixture) {},
			wantErr:   true,
		},
		"scan returns all .md sorted ascending": {
			setupFunc: func(f *Fixture) {
				f.SaveReport("2024-06-15", "## やった\nx\n")
				f.SaveReport("2024-05-30", "## やった\ny\n")
				f.SaveReport("2024-06-14", "## やった\nz\n")
			},
			wantDates: []string{"2024-05-30", "2024-06-14", "2024-06-15"},
		},
		"empty when no reports": {
			setupFunc: func(f *Fixture) {},
			wantDates: []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := New(t)
			tt.setupFunc(f)
			c := &migrateCmd{Date: tt.date}
			dates, err := c.findTargets(f.NewStorage())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got := make([]string, 0, len(dates))
			for _, d := range dates {
				got = append(got, d.Format("2006-01-02"))
			}
			assert.Equal(t, tt.wantDates, got)
		})
	}
}

// TestMigrateEmitReport は text / json の整形と stdout/stderr の分離を担保する。
func TestMigrateEmitReport(t *testing.T) {
	results := []migrationResult{
		{
			Date:     time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			Action:   migrateActionMigrate,
			Sections: []string{"done", "todo"},
		},
		{
			Date:            time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC),
			Action:          migrateActionSkip,
			Reason:          "already has .yaml",
			UnknownSections: []string{"foo"},
		},
		{
			Date:   time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			Action: migrateActionFail,
			Reason: "read legacy md: boom",
		},
	}

	tests := map[string]struct {
		apply        bool
		format       string
		wantStdout   []string // contains 部分一致
		wantStderr   []string
		wantNoStdout []string
	}{
		"dry-run text": {
			apply:  false,
			format: "text",
			wantStdout: []string{
				"[dry-run] would migrate 2024-06-15 (2 sections -> done/todo)",
				"[dry-run] would skip 2024-06-14 (already has .yaml)",
				"[dry-run] failed 2024-05-01 (read legacy md: boom)",
				"summary: would migrate 1 / skip 1 / fail 1 (total 3)",
			},
			wantStderr: []string{
				`warning: 2024-06-14: section "foo" has no matching question`,
			},
		},
		"apply text uses past tense": {
			apply:  true,
			format: "text",
			wantStdout: []string{
				"migrated 2024-06-15 (2 sections -> done/todo)",
				"skipped 2024-06-14 (already has .yaml)",
				"summary: migrated 1 / skip 1 / fail 1 (total 3)",
			},
			wantNoStdout: []string{"[dry-run]"},
		},
		"json dry-run": {
			apply:  false,
			format: "json",
			wantStdout: []string{
				`"mode": "dry-run"`,
				`"date": "2024-06-15"`,
				`"action": "migrate"`,
				`"action": "skip"`,
				`"action": "fail"`,
				`"migrated": 1`,
				`"skipped": 1`,
				`"failed": 1`,
				`"total": 3`,
			},
		},
		"json apply uses migrated action": {
			apply:  true,
			format: "json",
			wantStdout: []string{
				`"mode": "apply"`,
				`"action": "migrated"`,
			},
			wantNoStdout: []string{`"action": "migrate"`},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &migrateCmd{Apply: tt.apply, Format: tt.format}
			var stdout, stderr bytes.Buffer
			require.NoError(t, c.emitReport(&stdout, &stderr, results))
			for _, s := range tt.wantStdout {
				assert.Contains(t, stdout.String(), s, "stdout missing %q", s)
			}
			for _, s := range tt.wantStderr {
				assert.Contains(t, stderr.String(), s, "stderr missing %q", s)
			}
			for _, s := range tt.wantNoStdout {
				assert.NotContains(t, stdout.String(), s, "stdout must not contain %q", s)
			}
			if tt.format == "json" {
				var got migrationJSONReport
				require.NoError(t, json.Unmarshal(stdout.Bytes(), &got), "json must be parseable")
				assert.Equal(t, 3, got.Summary.Total)
			}
		})
	}
}

// TestMigrateRunIdempotent は同じ migrate を 2 回流しても結果が同じであることを担保する。
func TestMigrateRunIdempotent(t *testing.T) {
	f := New(t)
	cfg := &Config{
		Version:    1,
		StorageDir: f.TmpDir(),
		Questions: []QuestionConfig{
			{Key: "done", Label: "やった"},
		},
	}
	storage, err := NewStorage(cfg)
	require.NoError(t, err)
	defer storage.Close()

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	require.NoError(t, storage.SaveReport("## やった\nx\n", date))

	c := &migrateCmd{Apply: true}

	res1 := c.processOne(storage, cfg, date)
	assert.Equal(t, migrateActionMigrate, res1.Action)

	res2 := c.processOne(storage, cfg, date)
	assert.Equal(t, migrateActionSkip, res2.Action)
	assert.Equal(t, "already has .yaml", res2.Reason)

	// .yaml の中身は 2 回目で書き換わっていない (上書きなし) ことを確認する。
	got, err := storage.LoadReportStruct(date)
	require.NoError(t, err)
	assert.Equal(t, "x", got.Fields["done"].Body)
}
