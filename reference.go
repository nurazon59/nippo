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
	// 内側の `-->` で外側コメントが閉じてしまう (CommonMark の HTML block type 2)
	// のを防ぐため、終端マーカーのみ escape する
	content = strings.ReplaceAll(content, "-->", "--&gt;")
	return "<!--\n" + content + "\n-->"
}

// buildSameDayPreset は同日中の他フィールドを reference するための preset を返す。
// task_list 型 field も fieldBodyAsText 経由で bullet 化されるため、text/task_list が混在しても扱える。
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

	// 旧形式 (.md のみ) の過去日報は構造化 reference の対象外。
	// silent skip ではなく fs.ErrNotExist のみを許容し、他のエラーは伝搬する。
	prev, err := storage.LoadReportStruct(prevDate)
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

		refField, ok := prev.Fields[q.ReferenceKey]
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
