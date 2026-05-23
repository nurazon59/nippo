package main

import (
	"time"

	"github.com/nurazon59/nippo/renderer"
	"github.com/nurazon59/nippo/report"
)

// questionsToRendererQuestions は QuestionConfig の slice を renderer.Question slice に変換する。
// renderer は QuestionConfig に依存しないため、main 側で表示モデルへの adapter を持つ。
func questionsToRendererQuestions(qs []QuestionConfig) []renderer.Question {
	out := make([]renderer.Question, 0, len(qs))
	for _, q := range qs {
		out = append(out, renderer.Question{Key: q.Key, Label: q.Label})
	}
	return out
}

// newTextReport は 1 日分の構造化レポートを「全フィールド text 型」で初期化する。
// Step 7 で task_list 型が入るまでの暫定 helper だが、generate.go の意図を宣言的に保つ目的で関数化する。
func newTextReport(date time.Time, fields map[string]string) *report.Report {
	r := &report.Report{
		SchemaVersion: report.SupportedSchemaVersion,
		Date:          date,
		Fields:        make(map[string]report.FieldValue, len(fields)),
	}
	for k, v := range fields {
		r.Fields[k] = report.FieldValue{Type: report.FieldTypeText, Body: v}
	}
	return r
}
