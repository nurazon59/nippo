package main

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	done := make(chan struct{})
	var buf []byte
	go func() {
		buf, _ = io.ReadAll(r)
		close(done)
	}()

	fn()
	w.Close()
	os.Stdout = old
	<-done
	return string(buf)
}

func TestLatestCmdNoReports(t *testing.T) {
	tmp := t.TempDir()

	cfgPath := tmp + "/config.yaml"
	cfg := &Config{Version: 1, StorageDir: tmp}
	require.NoError(t, cfg.Save(cfgPath))

	oldConfig := CLI.Config
	CLI.Config = cfgPath
	defer func() { CLI.Config = oldConfig }()

	err := (&latestCmd{}).Run()
	require.Error(t, err)
}

func TestLatestCmdPicksNewest(t *testing.T) {
	tmp := t.TempDir()

	storage, err := NewStorage(&Config{StorageDir: tmp})
	require.NoError(t, err)
	require.NoError(t, storage.SaveReport("# older", time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)))
	require.NoError(t, storage.SaveReport("# newest", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)))
	require.NoError(t, storage.SaveReport("# middle", time.Date(2024, 6, 12, 0, 0, 0, 0, time.UTC)))

	cfgPath := tmp + "/config.yaml"
	cfg := &Config{Version: 1, StorageDir: tmp}
	require.NoError(t, cfg.Save(cfgPath))

	oldConfig := CLI.Config
	CLI.Config = cfgPath
	defer func() { CLI.Config = oldConfig }()

	out := captureStdout(t, func() {
		require.NoError(t, (&latestCmd{}).Run())
	})
	assert.Equal(t, "# newest", out)
}
