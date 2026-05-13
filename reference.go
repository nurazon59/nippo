package main

import (
	"errors"
	"io/fs"
	"strings"
	"time"
)

func commentOutPreset(content string) string {
	content = strings.TrimRight(content, "\r\n")
	if strings.TrimSpace(content) == "" {
		return ""
	}
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

func buildReferencePresets(storage *Storage, date time.Time, questions []QuestionConfig) (map[string]string, error) {
	previous, err := storage.LoadPreviousReport(date)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	labelByKey := make(map[string]string, len(questions))
	for _, q := range questions {
		labelByKey[q.Key] = q.Label
	}

	sections := extractReportSections(previous)
	presets := make(map[string]string)
	for _, q := range questions {
		if q.ReferenceKey == "" {
			continue
		}

		refLabel, ok := labelByKey[q.ReferenceKey]
		if !ok {
			continue
		}

		content, ok := sections[refLabel]
		if !ok {
			continue
		}

		preset := commentOutPreset(content)
		if preset == "" {
			continue
		}
		presets[q.Key] = preset
	}

	return presets, nil
}
