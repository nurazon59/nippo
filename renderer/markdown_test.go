package renderer_test

import (
	"testing"
	"time"

	"github.com/nurazon59/nippo/renderer"
	"github.com/nurazon59/nippo/report"
	"github.com/stretchr/testify/assert"
)

func TestMarkdown(t *testing.T) {
	date := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		report    *report.Report
		questions []renderer.Question
		want      string
	}{
		"all text fields (legacy compatible)": {
			report: &report.Report{
				SchemaVersion: report.SupportedSchemaVersion,
				Date:          date,
				Fields: map[string]report.FieldValue{
					"done":     {Type: report.FieldTypeText, Body: "実装した"},
					"todo":     {Type: report.FieldTypeText, Body: "レビューする"},
					"thoughts": {Type: report.FieldTypeText, Body: "順調"},
				},
			},
			questions: []renderer.Question{
				{Key: "done", Label: "やった"},
				{Key: "todo", Label: "やる"},
				{Key: "thoughts", Label: "所感"},
			},
			want: "# 日報 2026-05-23\n\n" +
				"## やった\n実装した\n" +
				"## やる\nレビューする\n" +
				"## 所感\n順調\n",
		},
		"task_list with full fields": {
			report: &report.Report{
				SchemaVersion: report.SupportedSchemaVersion,
				Date:          date,
				Fields: map[string]report.FieldValue{
					"tasks": {
						Type: report.FieldTypeTaskList,
						Tasks: []report.Task{
							{Title: "PR #47 review", Time: "1h30m", Outcome: "merged", Thoughts: "refactor しやすかった"},
						},
					},
				},
			},
			questions: []renderer.Question{
				{Key: "tasks", Label: "タスク"},
			},
			want: "# 日報 2026-05-23\n\n" +
				"## タスク\n" +
				"- PR #47 review (1h30m) merged\n" +
				"  refactor しやすかった\n",
		},
		"task_list with title only": {
			report: &report.Report{
				SchemaVersion: report.SupportedSchemaVersion,
				Date:          date,
				Fields: map[string]report.FieldValue{
					"tasks": {
						Type: report.FieldTypeTaskList,
						Tasks: []report.Task{
							{Title: "単発タスク"},
						},
					},
				},
			},
			questions: []renderer.Question{
				{Key: "tasks", Label: "タスク"},
			},
			want: "# 日報 2026-05-23\n\n" +
				"## タスク\n" +
				"- 単発タスク\n",
		},
		"task_list with time but no outcome": {
			report: &report.Report{
				SchemaVersion: report.SupportedSchemaVersion,
				Date:          date,
				Fields: map[string]report.FieldValue{
					"tasks": {
						Type: report.FieldTypeTaskList,
						Tasks: []report.Task{
							{Title: "調査", Time: "2h"},
						},
					},
				},
			},
			questions: []renderer.Question{
				{Key: "tasks", Label: "タスク"},
			},
			want: "# 日報 2026-05-23\n\n" +
				"## タスク\n" +
				"- 調査 (2h)\n",
		},
		"task_list with thoughts only": {
			report: &report.Report{
				SchemaVersion: report.SupportedSchemaVersion,
				Date:          date,
				Fields: map[string]report.FieldValue{
					"tasks": {
						Type: report.FieldTypeTaskList,
						Tasks: []report.Task{
							{Title: "もくもく", Thoughts: "集中できた"},
						},
					},
				},
			},
			questions: []renderer.Question{
				{Key: "tasks", Label: "タスク"},
			},
			want: "# 日報 2026-05-23\n\n" +
				"## タスク\n" +
				"- もくもく\n" +
				"  集中できた\n",
		},
		"task_list with outcome only": {
			report: &report.Report{
				SchemaVersion: report.SupportedSchemaVersion,
				Date:          date,
				Fields: map[string]report.FieldValue{
					"tasks": {
						Type: report.FieldTypeTaskList,
						Tasks: []report.Task{
							{Title: "対応", Outcome: "done"},
						},
					},
				},
			},
			questions: []renderer.Question{
				{Key: "tasks", Label: "タスク"},
			},
			want: "# 日報 2026-05-23\n\n" +
				"## タスク\n" +
				"- 対応 done\n",
		},
		"task_list zero tasks": {
			report: &report.Report{
				SchemaVersion: report.SupportedSchemaVersion,
				Date:          date,
				Fields: map[string]report.FieldValue{
					"tasks": {
						Type:  report.FieldTypeTaskList,
						Tasks: []report.Task{},
					},
				},
			},
			questions: []renderer.Question{
				{Key: "tasks", Label: "タスク"},
			},
			want: "# 日報 2026-05-23\n\n" +
				"## タスク\n",
		},
		"missing field renders empty section": {
			report: &report.Report{
				SchemaVersion: report.SupportedSchemaVersion,
				Date:          date,
				Fields:        map[string]report.FieldValue{},
			},
			questions: []renderer.Question{
				{Key: "done", Label: "やった"},
			},
			want: "# 日報 2026-05-23\n\n" +
				"## やった\n\n",
		},
		"questions order is preserved": {
			report: &report.Report{
				SchemaVersion: report.SupportedSchemaVersion,
				Date:          date,
				Fields: map[string]report.FieldValue{
					"a": {Type: report.FieldTypeText, Body: "A"},
					"b": {Type: report.FieldTypeText, Body: "B"},
					"c": {Type: report.FieldTypeText, Body: "C"},
				},
			},
			questions: []renderer.Question{
				{Key: "c", Label: "C ラベル"},
				{Key: "a", Label: "A ラベル"},
				{Key: "b", Label: "B ラベル"},
			},
			want: "# 日報 2026-05-23\n\n" +
				"## C ラベル\nC\n" +
				"## A ラベル\nA\n" +
				"## B ラベル\nB\n",
		},
		"mixed text and task_list": {
			report: &report.Report{
				SchemaVersion: report.SupportedSchemaVersion,
				Date:          date,
				Fields: map[string]report.FieldValue{
					"done": {
						Type: report.FieldTypeTaskList,
						Tasks: []report.Task{
							{Title: "実装", Time: "3h", Outcome: "完了"},
							{Title: "レビュー対応"},
						},
					},
					"thoughts": {Type: report.FieldTypeText, Body: "良い一日だった"},
				},
			},
			questions: []renderer.Question{
				{Key: "done", Label: "やった"},
				{Key: "thoughts", Label: "所感"},
			},
			want: "# 日報 2026-05-23\n\n" +
				"## やった\n" +
				"- 実装 (3h) 完了\n" +
				"- レビュー対応\n" +
				"## 所感\n良い一日だった\n",
		},
		"unknown type renders empty section": {
			report: &report.Report{
				SchemaVersion: report.SupportedSchemaVersion,
				Date:          date,
				Fields: map[string]report.FieldValue{
					"x": {Type: "weird"},
				},
			},
			questions: []renderer.Question{
				{Key: "x", Label: "X"},
			},
			want: "# 日報 2026-05-23\n\n" +
				"## X\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := renderer.Markdown(tt.report, tt.questions)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMarkdownReferenceBeforeHeading(t *testing.T) {
	tests := map[string]struct {
		body string
		want string
	}{
		"hook の後に見出しと本文を置く": {
			body: "<!--\nPR 一覧\n-->\n\n実装した",
			want: "<!--\nPR 一覧\n-->\n\n## やった\n実装した\n",
		},
		"複数の参考情報を元の順序で見出しの前に置く": {
			body: "<!--\n前日\n-->\n\n<!--\nPR 一覧\n-->\n\n実装した",
			want: "<!--\n前日\n-->\n\n<!--\nPR 一覧\n-->\n\n## やった\n実装した\n",
		},
		"参考情報だけでも見出しを残す": {
			body: "<!--\nPR 一覧\n-->",
			want: "<!--\nPR 一覧\n-->\n\n## やった\n\n",
		},
		"本文中のコメントは移動しない": {
			body: "実装した\n<!--\n補足\n-->",
			want: "## やった\n実装した\n<!--\n補足\n-->\n",
		},
		"閉じていないコメントは移動しない": {
			body: "<!--\n未完のコメント",
			want: "## やった\n<!--\n未完のコメント\n",
		},
		"先頭の空白と改行があっても参考情報を検出する": {
			body: "\r\n<!--\r\nPR 一覧\r\n-->\r\n\r\n実装した",
			want: "\r\n<!--\r\nPR 一覧\r\n-->\n\n## やった\n実装した\n",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := &report.Report{
				Date: time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC),
				Fields: map[string]report.FieldValue{
					"done": {Type: report.FieldTypeText, Body: tt.body},
				},
			}
			got := renderer.Markdown(r, []renderer.Question{{Key: "done", Label: "やった"}})
			assert.Equal(t, "# 日報 2026-05-23\n\n"+tt.want, got)
		})
	}
}
