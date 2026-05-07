package main

import (
	"bytes"
	"text/template"
	"time"
)

type Report struct {
	Date     time.Time
	Done     string
	Todo     string
	Thoughts string
}

var reportTmpl = template.Must(template.New("report").Parse(`# 日報 {{.Date}}

## やった
{{.Done}}
## やる
{{.Todo}}
## 所感
{{.Thoughts}}
`))

type templateData struct {
	Date     string
	Done     string
	Todo     string
	Thoughts string
}

func GenerateMarkdown(r *Report) string {
	var buf bytes.Buffer
	reportTmpl.Execute(&buf, templateData{
		Date:     r.Date.Format("2006-01-02"),
		Done:     r.Done,
		Todo:     r.Todo,
		Thoughts: r.Thoughts,
	})
	return buf.String()
}
