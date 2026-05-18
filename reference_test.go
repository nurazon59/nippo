package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentOutPreset(t *testing.T) {
	assert.Equal(t, "", commentOutPreset(""))
	assert.Equal(t, "<!--\nhello\nworld\n-->", commentOutPreset("hello\nworld"))
}

func TestBuildReferencePresets(t *testing.T) {
	tests := map[string]struct {
		questions   []QuestionConfig
		setupFunc   func(f *Fixture)
		wantPresets map[string]string
		wantAbsent  []string
	}{
		"with history": {
			questions: []QuestionConfig{
				{Key: "done", Label: "やった", Required: true},
				{Key: "todo", Label: "やる", Required: true, ReferenceKey: "done"},
				{Key: "thoughts", Label: "所感"},
			},
			setupFunc: func(f *Fixture) {
				f.SaveReport("2024-06-14", "# 日報 2024-06-14\n\n## やった\n古い作業\n")
				f.SaveReport("2024-06-15", "# 日報 2024-06-15\n\n## やった\n昨日の作業\n\n続き\n\n## やる\n次の作業\n")
			},
			wantPresets: map[string]string{
				"todo": "<!--\n昨日の作業\n\n続き\n-->",
			},
			wantAbsent: []string{"done"},
		},
		"without history": {
			questions: []QuestionConfig{
				{Key: "todo", Label: "やる", ReferenceKey: "done"},
			},
			setupFunc:   func(f *Fixture) {},
			wantPresets: map[string]string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := New(t)
			tt.setupFunc(f)

			targetDate := time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC)
			presets, err := buildReferencePresets(f.NewStorage(), targetDate, tt.questions)
			require.NoError(t, err)

			if len(tt.wantPresets) == 0 {
				assert.Empty(t, presets)
			} else {
				for k, v := range tt.wantPresets {
					assert.Equal(t, v, presets[k])
				}
			}
			for _, k := range tt.wantAbsent {
				_, ok := presets[k]
				assert.False(t, ok)
			}
		})
	}
}
