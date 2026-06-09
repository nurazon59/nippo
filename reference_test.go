package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nurazon59/nippo/report"
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
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := buildSameDayPreset(tt.answered, tt.question)
			assert.Equal(t, tt.want, got)
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
		"with history": {
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
			wantAbsent: []string{"done"},
		},
		"without history": {
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
					assert.Equal(t, v, presets[k])
				}
			}
			for _, k := range tt.wantAbsent {
				_, ok := presets[k]
				assert.False(t, ok)
			}
		})
	}
}
