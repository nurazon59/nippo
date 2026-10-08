package renderer

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/nurazon59/nippo/report"
)

type Question struct {
	Key   string
	Label string
}

func Markdown(r *report.Report, questions []Question) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# 日報 %s\n\n", r.Date.Format("2006-01-02"))
	for _, q := range questions {
		writeSection(&buf, q, r.Fields)
	}
	return buf.String()
}

func writeSection(buf *bytes.Buffer, q Question, fields map[string]report.FieldValue) {
	v, present := fields[q.Key]
	if !present {
		fmt.Fprintf(buf, "## %s\n\n", q.Label)
		return
	}
	switch v.Type {
	case report.FieldTypeText:
		references, body := splitLeadingReferences(v.Body)
		if references != "" {
			fmt.Fprintf(buf, "%s\n\n", references)
		}
		fmt.Fprintf(buf, "## %s\n%s\n", q.Label, body)
	case report.FieldTypeTaskList:
		fmt.Fprintf(buf, "## %s\n", q.Label)
		for _, t := range v.Tasks {
			writeTask(buf, t)
		}
	default:
		fmt.Fprintf(buf, "## %s\n", q.Label)
	}
}

// splitLeadingReferences は先頭の参考コメントだけを切り出し、本文中のコメントを維持する。
func splitLeadingReferences(body string) (string, string) {
	end := 0
	for {
		rest := strings.TrimLeft(body[end:], " \t\r\n")
		if !strings.HasPrefix(rest, "<!--") {
			break
		}
		closeIndex := strings.Index(rest, "-->")
		if closeIndex < 0 {
			break
		}
		end = len(body) - len(rest) + closeIndex + len("-->")
	}
	if end == 0 {
		return "", body
	}
	return body[:end], strings.TrimLeft(body[end:], "\r\n")
}

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
