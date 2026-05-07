package main

import (
	"bytes"
	"fmt"
	"time"
)

type Report struct {
	Date   time.Time
	Fields map[string]string
}

func GenerateMarkdown(r *Report, questions []QuestionConfig) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# 日報 %s\n\n", r.Date.Format("2006-01-02")))
	for _, q := range questions {
		buf.WriteString(fmt.Sprintf("## %s\n%s\n", q.Label, r.Fields[q.Key]))
	}
	return buf.String()
}
