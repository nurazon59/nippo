package main

import (
	"errors"
	"io/fs"
	"strings"
	"time"

	"github.com/nurazon59/nippo/report"
)

func commentOutPreset(content string) string {
	content = strings.TrimRight(content, "\r\n")
	if strings.TrimSpace(content) == "" {
		return ""
	}
	// 終端マーカーだけエスケープする
	content = strings.ReplaceAll(content, "-->", "--&gt;")
	return "<!--\n" + content + "\n-->"
}

func extractReportSections(content string) map[string]string {
	sections := make(map[string]string)

	var currentLabel string
	var lines []string
	flush := func() {
		if currentLabel == "" {
			return
		}
		sections[currentLabel] = strings.TrimRight(strings.Join(lines, "\n"), "\n")
		lines = nil
	}

	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			currentLabel = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}
		if currentLabel == "" {
			continue
		}
		lines = append(lines, line)
	}

	flush()
	return sections
}

func buildSameDayPreset(answered map[string]report.FieldValue, q QuestionConfig) string {
	if q.SameDayReferenceKey == "" {
		return ""
	}
	v, ok := answered[q.SameDayReferenceKey]
	if !ok {
		return ""
	}
	return commentOutPreset(fieldBodyAsText(v))
}

func buildReferencePresets(storage *Storage, date time.Time, questions []QuestionConfig) (map[string]string, error) {
	prevDate, err := storage.LoadPreviousReport(date)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	previous, err := storage.LoadReportStruct(prevDate)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	presets := make(map[string]string)
	for _, q := range questions {
		if q.ReferenceKey == "" {
			continue
		}

		refField, ok := previous.Fields[q.ReferenceKey]
		if !ok {
			continue
		}

		body := fieldBodyAsText(refField)
		if body == "" {
			continue
		}

		preset := commentOutPreset(body)
		if preset == "" {
			continue
		}
		presets[q.Key] = preset
	}

	return presets, nil
}
