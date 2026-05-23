package main

import (
	"testing"
	"time"

	"github.com/nurazon59/nippo/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentOutPreset(t *testing.T) {
	tests := map[string]struct {
		content string
		want    string
	}{
		"empty":             {content: "", want: ""},
		"plain multiline":   {content: "hello\nworld", want: "<!--\nhello\nworld\n-->"},
		"nested terminator": {content: "前段\n<!--\nhook\n-->", want: "<!--\n前段\n<!--\nhook\n--&gt;\n-->"},
		"trailing newline":  {content: "x\n", want: "<!--\nx\n-->"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, commentOutPreset(tt.content))
		})
	}
}

func TestBuildSameDayPreset(t *testing.T) {
	tests := map[string]struct {
		question QuestionConfig
		answered map[string]report.FieldValue
		want     string
	}{
		"no reference key": {
			question: QuestionConfig{Key: "todo"},
			answered: map[string]report.FieldValue{"done": {Type: report.FieldTypeText, Body: "なにかやった"}},
			want:     "",
		},
		"missing answer": {
			question: QuestionConfig{Key: "todo", SameDayReferenceKey: "done"},
			answered: map[string]report.FieldValue{},
			want:     "",
		},
		"empty answer": {
			question: QuestionConfig{Key: "todo", SameDayReferenceKey: "done"},
			answered: map[string]report.FieldValue{"done": {Type: report.FieldTypeText, Body: ""}},
			want:     "",
		},
		"plain answer": {
			question: QuestionConfig{Key: "todo", SameDayReferenceKey: "done"},
			answered: map[string]report.FieldValue{"done": {Type: report.FieldTypeText, Body: "昨日の続きをやった"}},
			want:     "<!--\n昨日の続きをやった\n-->",
		},
		"answer with nested hook comment": {
			question: QuestionConfig{Key: "todo", SameDayReferenceKey: "done"},
			answered: map[string]report.FieldValue{"done": {Type: report.FieldTypeText, Body: "前段\n<!--\nhook\n-->"}},
			want:     "<!--\n前段\n<!--\nhook\n--&gt;\n-->",
		},
		"task_list answer は bullet 形式で引き継ぐ": {
			question: QuestionConfig{Key: "thoughts", SameDayReferenceKey: "done"},
			answered: map[string]report.FieldValue{"done": {
				Type: report.FieldTypeTaskList,
				Tasks: []report.Task{
					{Title: "設計レビュー", Time: "30m", Outcome: "done"},
					{Title: "実装"},
				},
			}},
			want: "<!--\n- 設計レビュー (30m) done\n- 実装\n-->",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := buildSameDayPreset(tt.answered, tt.question)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFieldBodyAsText(t *testing.T) {
	tests := map[string]struct {
		value report.FieldValue
		want  string
	}{
		"text 型はそのまま body を返す": {
			value: report.FieldValue{Type: report.FieldTypeText, Body: "昨日の作業\n\n続き"},
			want:  "昨日の作業\n\n続き",
		},
		"text 型で空 body は空文字": {
			value: report.FieldValue{Type: report.FieldTypeText, Body: ""},
			want:  "",
		},
		"task_list は bullet で並べる": {
			value: report.FieldValue{
				Type: report.FieldTypeTaskList,
				Tasks: []report.Task{
					{Title: "A", Time: "1h", Outcome: "done", Thoughts: "順調"},
					{Title: "B"},
				},
			},
			want: "- A (1h) done\n  順調\n- B",
		},
		"task_list が空なら空文字": {
			value: report.FieldValue{Type: report.FieldTypeTaskList, Tasks: []report.Task{}},
			want:  "",
		},
		"未知の type は空文字": {
			value: report.FieldValue{Type: "weird", Body: "x"},
			want:  "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, fieldBodyAsText(tt.value))
		})
	}
}

func TestBuildReferencePresets(t *testing.T) {
	tests := map[string]struct {
		questions   []QuestionConfig
		setupFunc   func(f *Fixture)
		wantPresets map[string]string
		wantAbsent  []string
	}{
		"前日の text reference を引き継ぐ": {
			questions: []QuestionConfig{
				{Key: "done", Label: "やった", Required: true},
				{Key: "todo", Label: "やる", Required: true, ReferenceKey: "done"},
				{Key: "thoughts", Label: "所感"},
			},
			setupFunc: func(f *Fixture) {
				f.SaveReportStruct("2024-06-14", map[string]report.FieldValue{
					"done": {Type: report.FieldTypeText, Body: "古い作業"},
				})
				f.SaveReportStruct("2024-06-15", map[string]report.FieldValue{
					"done": {Type: report.FieldTypeText, Body: "昨日の作業\n\n続き"},
					"todo": {Type: report.FieldTypeText, Body: "次の作業"},
				})
			},
			wantPresets: map[string]string{
				"todo": "<!--\n昨日の作業\n\n続き\n-->",
			},
			wantAbsent: []string{"done", "thoughts"},
		},
		"前日の task_list reference を bullet 形式で引き継ぐ": {
			questions: []QuestionConfig{
				{Key: "done", Label: "やった"},
				{Key: "todo", Label: "やる", ReferenceKey: "done"},
			},
			setupFunc: func(f *Fixture) {
				f.SaveReportStruct("2024-06-15", map[string]report.FieldValue{
					"done": {
						Type: report.FieldTypeTaskList,
						Tasks: []report.Task{
							{Title: "設計レビュー", Time: "30m", Outcome: "done", Thoughts: "懸念点を共有"},
							{Title: "実装"},
						},
					},
				})
			},
			wantPresets: map[string]string{
				"todo": "<!--\n- 設計レビュー (30m) done\n  懸念点を共有\n- 実装\n-->",
			},
			wantAbsent: []string{"done"},
		},
		"ReferenceKey が空ならスキップ": {
			questions: []QuestionConfig{
				{Key: "done", Label: "やった"},
				{Key: "todo", Label: "やる"},
			},
			setupFunc: func(f *Fixture) {
				f.SaveReportStruct("2024-06-15", map[string]report.FieldValue{
					"done": {Type: report.FieldTypeText, Body: "x"},
				})
			},
			wantPresets: map[string]string{},
		},
		"未知の ReferenceKey はスキップ": {
			questions: []QuestionConfig{
				{Key: "todo", Label: "やる", ReferenceKey: "missing"},
			},
			setupFunc: func(f *Fixture) {
				f.SaveReportStruct("2024-06-15", map[string]report.FieldValue{
					"done": {Type: report.FieldTypeText, Body: "x"},
				})
			},
			wantPresets: map[string]string{},
		},
		"前日が空 body ならスキップ": {
			questions: []QuestionConfig{
				{Key: "done", Label: "やった"},
				{Key: "todo", Label: "やる", ReferenceKey: "done"},
			},
			setupFunc: func(f *Fixture) {
				f.SaveReportStruct("2024-06-15", map[string]report.FieldValue{
					"done": {Type: report.FieldTypeText, Body: ""},
				})
			},
			wantPresets: map[string]string{},
		},
		"前日が legacy .md のみで構造化なしなら preset は空": {
			questions: []QuestionConfig{
				{Key: "done", Label: "やった"},
				{Key: "todo", Label: "やる", ReferenceKey: "done"},
			},
			setupFunc: func(f *Fixture) {
				f.SaveReport("2024-06-15", "# 日報 2024-06-15\n\n## やった\n昨日の作業\n")
			},
			wantPresets: map[string]string{},
		},
		"前日がなければ preset は空": {
			questions: []QuestionConfig{
				{Key: "todo", Label: "やる", ReferenceKey: "done"},
			},
			setupFunc:   func(f *Fixture) {},
			wantPresets: map[string]string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := New(t)
			tt.setupFunc(f)

			targetDate := time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC)
			presets, err := buildReferencePresets(f.NewStorage(), targetDate, tt.questions)
			require.NoError(t, err)

			if len(tt.wantPresets) == 0 {
				assert.Empty(t, presets)
			} else {
				for k, v := range tt.wantPresets {
					assert.Equal(t, v, presets[k], "key=%s", k)
				}
			}
			for _, k := range tt.wantAbsent {
				_, ok := presets[k]
				assert.False(t, ok, "key=%s", k)
			}
		})
	}
}
