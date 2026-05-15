package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditCmdInvalidDate(t *testing.T) {
	cmd := &editCmd{Date: "invalid"}
	err := cmd.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date format")
}

func TestEditCmdReportNotFound(t *testing.T) {
	tmp := t.TempDir()

	cfgPath := tmp + "/config.yaml"
	cfg := &Config{
		Version:    1,
		StorageDir: tmp,
		Questions:  []QuestionConfig{{Key: "done", Label: "Done", Required: true}},
	}
	err := cfg.Save(cfgPath)
	require.NoError(t, err)

	oldConfig := CLI.Config
	CLI.Config = cfgPath
	defer func() { CLI.Config = oldConfig }()

	cmd := &editCmd{
		Date: "2024-06-15",
		openEditor: func(cmd, content string) (string, error) {
			return content, nil
		},
	}
	err = cmd.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report not found")
}

func TestEditCmdNoChanges(t *testing.T) {
	tmp := t.TempDir()
	storage, err := NewStorage(tmp)
	require.NoError(t, err)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	content := "# test report"
	err = storage.Save(content, date)
	require.NoError(t, err)

	cfgPath := tmp + "/config.yaml"
	cfg := &Config{
		Version:    1,
		StorageDir: tmp,
		Questions:  []QuestionConfig{{Key: "done", Label: "Done", Required: true}},
	}
	err = cfg.Save(cfgPath)
	require.NoError(t, err)

	oldConfig := CLI.Config
	CLI.Config = cfgPath
	defer func() { CLI.Config = oldConfig }()

	cmd := &editCmd{
		Date: "2024-06-15",
		openEditor: func(cmd, content string) (string, error) {
			return content, nil
		},
	}
	err = cmd.Run()
	require.NoError(t, err)
}
