package main

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTUIStore struct {
	paths    []string
	contents map[string]string
	err      error
}

func (f fakeTUIStore) ListReports() ([]string, error) {
	return f.paths, f.err
}

func (f fakeTUIStore) LoadReportMarkdown(date time.Time, _ []QuestionConfig) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.contents[date.Format("2006-01-02")], nil
}

func TestTUIDatesFromPaths(t *testing.T) {
	tests := map[string]struct {
		paths []string
		want  []string
	}{
		"日付の降順で重複を除去": {
			paths: []string{"2026/10/28.md", "2026/10/30.md", "2026/10/30.md"},
			want:  []string{"2026-10-30", "2026-10-28"},
		},
		"不正なパスを無視": {
			paths: []string{"invalid", "../../2026/10/30.md", "2026/13/01.md"},
			want:  []string{},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := datesFromPaths(tt.paths)
			values := make([]string, 0, len(got))
			for _, d := range got {
				values = append(values, d.Format("2006-01-02"))
			}
			assert.Equal(t, tt.want, values)
		})
	}
}

func TestTUIFiltering(t *testing.T) {
	tests := map[string]struct {
		query string
		want  string
		count int
	}{
		"日付検索": {query: "2026-10-30", want: "2026-10-30", count: 1},
		"本文検索は大文字小文字を無視": {query: "oauth", want: "2026-10-29", count: 1},
		"該当なし":   {query: "nothing", want: "", count: 0},
		"空文字は全件": {query: "", want: "2026-10-30", count: 2},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			store := fakeTUIStore{
				paths: []string{"2026/10/29.md", "2026/10/30.md"},
				contents: map[string]string{
					"2026-10-29": "OAuth の実装",
					"2026-10-30": "Go の調査",
				},
			}
			model := newTUIModel(store, nil)
			require.NoError(t, model.reload(""))
			model.query = tt.query
			model.refilter("")
			assert.Len(t, model.filtered, tt.count)
			assert.Equal(t, tt.want, model.selectedDate())
		})
	}
}

func TestTUIReloadErrors(t *testing.T) {
	tests := map[string]struct {
		store   fakeTUIStore
		wantErr bool
	}{
		"読み込み成功": {
			store: fakeTUIStore{paths: []string{"2026/10/30.md"}, contents: map[string]string{"2026-10-30": "日報"}},
		},
		"ストレージのエラーを返す": {
			store:   fakeTUIStore{err: errors.New("failed")},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			m := newTUIModel(tt.store, nil)
			err := m.reload("")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
