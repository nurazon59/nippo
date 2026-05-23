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

func buildSameDayPreset(answered map[string]string, q QuestionConfig) string {
	if q.SameDayReferenceKey == "" {
		return ""
	}
	content, ok := answered[q.SameDayReferenceKey]
	if !ok {
		return ""
	}
	return commentOutPreset(content)
}

// fieldBodyAsText は preset 用に FieldValue を平文化する。
// reference は「前日内容を当日エディタの初期値として渡す」用途のため、
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
