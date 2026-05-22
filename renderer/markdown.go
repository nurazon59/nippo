// Package renderer は構造化スキーマ v1 の Report を Markdown 文字列へ変換する純関数群を提供する。
// main package の QuestionConfig を直接参照しないことで、IR (report) と CLI 設定の循環依存を断つ。
package renderer

import (
	"bytes"
	"fmt"

	"github.com/nurazon59/nippo/report"
)

// Question は renderer が必要とする最小情報のみを持つ表示モデル。
// main 側で QuestionConfig からの adapter を書き、renderer は CLI 設定に依存しない。
type Question struct {
	Key   string
	Label string
}

// Markdown は Report と Questions から日報 Markdown を組み立てる。
// 出力順は questions スライスの並びに従う。Fields に key が無い場合は legacy 互換の空 text セクションを出す。
func Markdown(r *report.Report, questions []Question) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# 日報 %s\n\n", r.Date.Format("2006-01-02"))
	for _, q := range questions {
		writeSection(&buf, q, r.Fields[q.Key], hasField(r.Fields, q.Key))
	}
	return buf.String()
}

// hasField は map にキーが存在するかを返す。
// 「未指定」と「空 text」を区別するためにゼロ値判定ではなく ok を見る必要がある。
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
		// Step 1 の unmarshal で reject 済みのはずだが、defensive にヘッダだけ出して落ちないようにする。
		fmt.Fprintf(buf, "## %s\n", q.Label)
	}
}

// writeTask は task_list の 1 要素をフォーマットする。
// time/outcome/thoughts の空判定で区切り (スペース/改行) を含めて省略する。
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
