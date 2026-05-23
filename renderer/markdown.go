// Package renderer は構造化スキーマ v1 の Report を Markdown に変換する純関数を提供する。
package renderer

import (
	"bytes"
	"fmt"

	"github.com/nurazon59/nippo/report"
)

// Question は表示に必要な最小情報を表す。
type Question struct {
	Key   string
	Label string
}

// Markdown は Report と Questions から日報 Markdown を組み立てる。
func Markdown(r *report.Report, questions []Question) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# 日報 %s\n\n", r.Date.Format("2006-01-02"))
	for _, q := range questions {
		writeSection(&buf, q, r.Fields[q.Key], hasField(r.Fields, q.Key))
	}
	return buf.String()
}

// hasField は map にキーが存在するかを返す。
func hasField(fields map[string]report.FieldValue, key string) bool {
	_, ok := fields[key]
	return ok
}

// writeSection は 1 つの質問に対応するセクションを書き出す。
// missing は legacy GenerateMarkdown 互換のため `## label\n\n` を出す。
func writeSection(buf *bytes.Buffer, q Question, v report.FieldValue, present bool) {
	if !present {
		fmt.Fprintf(buf, "## %s\n\n", q.Label)
		return
	}
	switch v.Type {
	case report.FieldTypeText:
		fmt.Fprintf(buf, "## %s\n%s\n", q.Label, v.Body)
	case report.FieldTypeTaskList:
		fmt.Fprintf(buf, "## %s\n", q.Label)
		for _, t := range v.Tasks {
			writeTask(buf, t)
		}
	default:
		// unknown type の場合はヘッダのみ出す。
		fmt.Fprintf(buf, "## %s\n", q.Label)
	}
}

// writeTask は task_list の要素をフォーマットする。
func writeTask(buf *bytes.Buffer, t report.Task) {
	buf.WriteString("- ")
	buf.WriteString(t.Title)
	if t.Time != "" {
		fmt.Fprintf(buf, " (%s)", t.Time)
	}
	if t.Outcome != "" {
		buf.WriteString(" ")
		buf.WriteString(t.Outcome)
	}
	buf.WriteString("\n")
	if t.Thoughts != "" {
		buf.WriteString("  ")
		buf.WriteString(t.Thoughts)
		buf.WriteString("\n")
	}
}
