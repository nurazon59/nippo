package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShowCmd は show が .yaml ベースで render する経路と、legacy .md fallback の双方を
// e2e で担保する。.yaml が存在する場合は renderer.Markdown が canonical な出力源となる。
func TestShowCmd(t *testing.T) {
	tests := map[string]struct {
		date    string
		setup   func(t *testing.T, storage *Storage)
		wantErr string
		wantOut string
	}{
		"yaml-backed report": {
			date: "2024-06-15",
			setup: func(t *testing.T, storage *Storage) {
				d := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
				r := newTextReport(d, map[string]string{"done": "Aした", "todo": "Bする"})
				require.NoError(t, storage.SaveReportStruct(r))
			},
			wantOut: "# 日報 2024-06-15\n\n## やった\nAした\n## やる\nBする\n",
		},
		"legacy md fallback": {
			date: "2024-06-15",
			setup: func(t *testing.T, storage *Storage) {
				d := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
				require.NoError(t, storage.SaveReport("# legacy md\n", d))
			},
			wantOut: "# legacy md\n",
		},
		"missing report errors": {
			date:    "2024-06-15",
			setup:   func(t *testing.T, storage *Storage) {},
			wantErr: "no report found for 2024-06-15",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()

			storage, err := NewStorage(&Config{StorageDir: tmp})
			require.NoError(t, err)
			tt.setup(t, storage)

			cfgPath := tmp + "/config.yaml"
			cfg := &Config{
				Version:    1,
				StorageDir: tmp,
				Questions: []QuestionConfig{
					{Key: "done", Label: "やった"},
					{Key: "todo", Label: "やる"},
				},
			}
			require.NoError(t, cfg.Save(cfgPath))

			oldConfig := CLI.Config
			CLI.Config = cfgPath
			defer func() { CLI.Config = oldConfig }()

			var runErr error
			out := captureStdout(t, func() {
				runErr = (&showCmd{Date: tt.date}).Run()
			})

			if tt.wantErr != "" {
				require.Error(t, runErr)
				assert.Contains(t, runErr.Error(), tt.wantErr)
				return
			}
			require.NoError(t, runErr)
			assert.Equal(t, tt.wantOut, out)
		})
	}
}
