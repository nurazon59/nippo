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

func TestLatestCmd(t *testing.T) {
	type save struct {
		date    string
		content string
	}

	tests := map[string]struct {
		saves      []save
		wantErr    bool
		wantStdout string
	}{
		"no reports errors": {
			wantErr: true,
		},
		"single entry": {
			saves:      []save{{"2024-06-15", "# only"}},
			wantStdout: "# only",
		},
		"picks newest among many": {
			saves: []save{
				{"2024-06-10", "# older"},
				{"2024-06-15", "# newest"},
				{"2024-06-12", "# middle"},
			},
			wantStdout: "# newest",
		},
	}

	parseDate := func(t *testing.T, s string) time.Time {
		t.Helper()
		d, err := time.Parse("2006-01-02", s)
		require.NoError(t, err)
		return d
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()

			if len(tt.saves) > 0 {
				storage, err := NewStorage(&Config{StorageDir: tmp})
				require.NoError(t, err)
				for _, s := range tt.saves {
					require.NoError(t, storage.SaveReport(s.content, parseDate(t, s.date)))
				}
			}

			cfgPath := tmp + "/config.yaml"
			cfg := &Config{Version: 1, StorageDir: tmp}
			require.NoError(t, cfg.Save(cfgPath))

			oldConfig := CLI.Config
			CLI.Config = cfgPath
			defer func() { CLI.Config = oldConfig }()

			var runErr error
			out := captureStdout(t, func() {
				runErr = (&latestCmd{}).Run()
			})

			if tt.wantErr {
				require.Error(t, runErr)
				return
			}
			require.NoError(t, runErr)
			assert.Equal(t, tt.wantStdout, out)
		})
	}
}
