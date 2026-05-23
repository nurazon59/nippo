package main

import (
	"strings"

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

// fieldBodyAsText は preset 用に FieldValue を平文化する。
// reference / same-day reference は「過去内容を当日エディタの初期値として渡す」用途のため、
// task_list は - title (time) outcome / thoughts の bullet 形式に変換する。
func fieldBodyAsText(v report.FieldValue) string {
	switch v.Type {
	case report.FieldTypeText:
		return v.Body
	case report.FieldTypeTaskList:
		if len(v.Tasks) == 0 {
			return ""
		}
		var lines []string
		for _, t := range v.Tasks {
			head := "- " + t.Title
			if t.Time != "" {
				head += " (" + t.Time + ")"
			}
			if t.Outcome != "" {
				head += " " + t.Outcome
			}
			lines = append(lines, head)
			if t.Thoughts != "" {
				lines = append(lines, "  "+t.Thoughts)
			}
		}
		return strings.Join(lines, "\n")
	}
	return ""
}
