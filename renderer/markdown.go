package renderer

import (
	"bytes"
	"fmt"

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
		v, ok := r.Fields[q.Key]
		writeSection(&buf, q, v, ok)
	}
	return buf.String()
}

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
		fmt.Fprintf(buf, "## %s\n", q.Label)
	}
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
