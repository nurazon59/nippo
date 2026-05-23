package main

import (
	"testing"
	"time"

	"github.com/nurazon59/nippo/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditCmdInvalidDate(t *testing.T) {
	cmd := &editCmd{Date: "invalid"}
	err := cmd.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date format")
}

// TestExistingFieldHelpers は edit フォーム再開モデルの「default 値流し込み」ロジックを担保する。
// runForm 本体は survey 依存で interactive テストできないため、ディスパッチ前の helper を直接呼ぶ。
func TestExistingFieldHelpers(t *testing.T) {
	textInitial := &report.Report{
		SchemaVersion: report.SupportedSchemaVersion,
		Date:          time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		Fields: map[string]report.FieldValue{
			"done":     {Type: report.FieldTypeText, Body: "やったこと本文"},
			"tasks":    {Type: report.FieldTypeTaskList, Tasks: []report.Task{{Title: "A"}, {Title: "B", Time: "30m"}}},
			"empty":    {Type: report.FieldTypeText, Body: ""},
			"emptylst": {Type: report.FieldTypeTaskList, Tasks: []report.Task{}},
		},
	}

	textTests := map[string]struct {
		initial *report.Report
		key     string
		want    string
	}{
		"nil initial は空文字 (新規 generate と同等)": {
			initial: nil,
			key:     "done",
			want:    "",
		},
		"text 型既存値は body を返す": {
			initial: textInitial,
			key:     "done",
			want:    "やったこと本文",
		},
		"key 不在は空文字": {
			initial: textInitial,
			key:     "missing",
			want:    "",
		},
		"type 不一致 (task_list を text として参照) は空文字": {
			initial: textInitial,
			key:     "tasks",
			want:    "",
		},
		"text 型でも body 空なら空文字": {
			initial: textInitial,
			key:     "empty",
			want:    "",
		},
	}
	for name, tt := range textTests {
		t.Run("text/"+name, func(t *testing.T) {
			got := existingTextBody(tt.initial, tt.key)
			assert.Equal(t, tt.want, got)
		})
	}

	taskTests := map[string]struct {
		initial *report.Report
		key     string
		want    []report.Task
	}{
		"nil initial は nil (新規 generate と同等)": {
			initial: nil,
			key:     "tasks",
			want:    nil,
		},
		"task_list 既存値は Tasks を返す": {
			initial: textInitial,
			key:     "tasks",
			want:    []report.Task{{Title: "A"}, {Title: "B", Time: "30m"}},
		},
		"key 不在は nil": {
			initial: textInitial,
			key:     "missing",
			want:    nil,
		},
		"type 不一致 (text を task_list として参照) は nil": {
			initial: textInitial,
			key:     "done",
			want:    nil,
		},
		"task_list で 0 件は空 slice をそのまま返す": {
			initial: textInitial,
			key:     "emptylst",
			want:    []report.Task{},
		},
	}
	for name, tt := range taskTests {
		t.Run("task_list/"+name, func(t *testing.T) {
			got := existingTasks(tt.initial, tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestEditCmdRoundTripOnMissingYaml は「.yaml が無い date を edit しようとした場合に
// 新規 Report として扱える状態 (= existingTextBody / existingTasks が空を返す)」までを最小限担保する。
// runForm 本体は interactive のため呼ばないが、storage.LoadReportStruct の fs.ErrNotExist 経路で
// edit.go が panic しないことは edit.Run の早期 return (date parse / cfg load) でカバー済み。
func TestEditCmdRoundTripOnMissingYaml(t *testing.T) {
	tmp := t.TempDir()
	cfg := &Config{
		Version:    1,
		StorageDir: tmp,
		Questions:  []QuestionConfig{{Key: "done", Label: "Done", Required: true}},
	}
	storage, err := NewStorage(cfg)
	require.NoError(t, err)
	defer storage.Close()

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	// .yaml 不在の date を LoadReportStruct した結果は fs.ErrNotExist となり、
	// edit.go は existing=nil で runForm に進む。本テストでは existing=nil 経路の helper を検証する。
	_, err = storage.LoadReportStruct(date)
	require.Error(t, err)

	assert.Equal(t, "", existingTextBody(nil, "done"))
	assert.Nil(t, existingTasks(nil, "done"))
}

// TestEditCmdRoundTripOnExistingYaml は「既存 .yaml を Load → existing helper で値が引ける」
// までを round-trip で担保する。runForm 本体は呼ばないが、edit が頼る読み出し経路が壊れないことを示す。
func TestEditCmdRoundTripOnExistingYaml(t *testing.T) {
	tmp := t.TempDir()
	cfg := &Config{
		Version:    1,
		StorageDir: tmp,
		Questions: []QuestionConfig{
			{Key: "done", Label: "Done", Required: true},
			{Key: "tasks", Label: "Tasks", Type: "task_list"},
		},
	}
	storage, err := NewStorage(cfg)
	require.NoError(t, err)
	defer storage.Close()

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	original := &report.Report{
		SchemaVersion: report.SupportedSchemaVersion,
		Date:          date,
		Fields: map[string]report.FieldValue{
			"done":  {Type: report.FieldTypeText, Body: "保存済み本文"},
			"tasks": {Type: report.FieldTypeTaskList, Tasks: []report.Task{{Title: "T1"}}},
		},
	}
	require.NoError(t, storage.SaveReportStruct(original))

	loaded, err := storage.LoadReportStruct(date)
	require.NoError(t, err)

	assert.Equal(t, "保存済み本文", existingTextBody(loaded, "done"))
	assert.Equal(t, []report.Task{{Title: "T1"}}, existingTasks(loaded, "tasks"))
}
