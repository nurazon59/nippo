package main

import (
	"testing"
	"time"

	"github.com/nurazon59/nippo/report"
	"github.com/stretchr/testify/assert"
)

// TestApplyAnswer は type ごとの FieldValue 組み立てロジックを単体で担保する。
// runForm 本体は survey 依存で interactive テストできないため、ディスパッチで使う純粋関数を直接呼ぶ。
func TestApplyAnswer(t *testing.T) {
	tests := map[string]struct {
		setup func(r *report.Report)
		want  report.FieldValue
		key   string
	}{
		"text 型はそのまま body を詰める": {
			setup: func(r *report.Report) {
				applyTextAnswer(r, QuestionConfig{Key: "done"}, "やったこと")
			},
			key:  "done",
			want: report.FieldValue{Type: report.FieldTypeText, Body: "やったこと"},
		},
		"text 型は空 body でも明示的に Type=text を入れる": {
			setup: func(r *report.Report) {
				applyTextAnswer(r, QuestionConfig{Key: "thoughts"}, "")
			},
			key:  "thoughts",
			want: report.FieldValue{Type: report.FieldTypeText, Body: ""},
		},
		"task_list 型は Tasks を詰める": {
			setup: func(r *report.Report) {
				applyTaskListAnswer(r, QuestionConfig{Key: "done", Type: "task_list"}, []report.Task{
					{Title: "A", Time: "30m"},
					{Title: "B"},
				})
			},
			key: "done",
			want: report.FieldValue{
				Type: report.FieldTypeTaskList,
				Tasks: []report.Task{
					{Title: "A", Time: "30m"},
					{Title: "B"},
				},
			},
		},
		"task_list 型は nil tasks を空 slice に正規化する": {
			setup: func(r *report.Report) {
				applyTaskListAnswer(r, QuestionConfig{Key: "done", Type: "task_list"}, nil)
			},
			key: "done",
			want: report.FieldValue{
				Type:  report.FieldTypeTaskList,
				Tasks: []report.Task{},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := &report.Report{
				SchemaVersion: report.SupportedSchemaVersion,
				Date:          time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
				Fields:        map[string]report.FieldValue{},
			}
			tt.setup(r)
			got, ok := r.Fields[tt.key]
			assert.True(t, ok, "key=%s should be set", tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}
