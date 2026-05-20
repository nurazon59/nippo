package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunHooks(t *testing.T) {
	date := time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		hooks      []HookConfig
		wantKeys   map[string]string
		absentKeys []string
		maxElapsed time.Duration
	}{
		"captures stdout into commented preset": {
			hooks: []HookConfig{
				{Name: "echo", Command: "echo from-hook", Keys: []string{"done"}},
			},
			wantKeys: map[string]string{"done": "<!--\nfrom-hook\n-->"},
		},
		"single hook fans out to multiple keys": {
			hooks: []HookConfig{
				{Name: "echo", Command: "echo same", Keys: []string{"done", "todo"}},
			},
			wantKeys: map[string]string{
				"done": "<!--\nsame\n-->",
				"todo": "<!--\nsame\n-->",
			},
		},
		"timeout drops entry but other hooks survive": {
			hooks: []HookConfig{
				{Name: "slow", Command: "sleep 5", Keys: []string{"done"}, Timeout: "100ms"},
				{Name: "fast", Command: "echo fast", Keys: []string{"todo"}},
			},
			wantKeys:   map[string]string{"todo": "<!--\nfast\n-->"},
			absentKeys: []string{"done"},
		},
		"non-zero exit drops entry but other hook still runs": {
			hooks: []HookConfig{
				{Name: "fail", Command: "exit 1", Keys: []string{"done"}},
				{Name: "ok", Command: "echo ok", Keys: []string{"todo"}},
			},
			wantKeys:   map[string]string{"todo": "<!--\nok\n-->"},
			absentKeys: []string{"done"},
		},
		"hooks run in parallel": {
			hooks: []HookConfig{
				{Name: "a", Command: "sleep 1 && echo a", Keys: []string{"done"}},
				{Name: "b", Command: "sleep 1 && echo b", Keys: []string{"todo"}},
			},
			wantKeys: map[string]string{
				"done": "<!--\na\n-->",
				"todo": "<!--\nb\n-->",
			},
			maxElapsed: 1800 * time.Millisecond,
		},
		"NIPPO_DATE env var is exposed": {
			hooks: []HookConfig{
				{Name: "env", Command: `printf "%s" "$NIPPO_DATE"`, Keys: []string{"done"}},
			},
			wantKeys: map[string]string{"done": "<!--\n2024-06-16\n-->"},
		},
		"multiple hooks on same key concatenate in declaration order": {
			hooks: []HookConfig{
				{Name: "first", Command: "echo one", Keys: []string{"done"}},
				{Name: "second", Command: "echo two", Keys: []string{"done"}},
			},
			wantKeys: map[string]string{"done": "<!--\none\n-->\n\n<!--\ntwo\n-->"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			out := RunHooks(context.Background(), tt.hooks, date)
			elapsed := time.Since(start)

			for k, want := range tt.wantKeys {
				assert.Equal(t, want, out[k], "key=%s", k)
			}
			for _, k := range tt.absentKeys {
				_, ok := out[k]
				assert.False(t, ok, "key %q should be absent", k)
			}
			if tt.maxElapsed > 0 {
				assert.Less(t, elapsed, tt.maxElapsed)
			}
		})
	}
}

func TestMergePresets(t *testing.T) {
	tests := map[string]struct {
		presets map[string]string
		hookOut map[string]string
		want    map[string]string
	}{
		"both present concatenated": {
			presets: map[string]string{"done": "<!--\nprev\n-->"},
			hookOut: map[string]string{"done": "<!--\nhook\n-->"},
			want:    map[string]string{"done": "<!--\nprev\n-->\n\n<!--\nhook\n-->"},
		},
		"only hook output": {
			presets: map[string]string{},
			hookOut: map[string]string{"done": "<!--\nhook\n-->"},
			want:    map[string]string{"done": "<!--\nhook\n-->"},
		},
		"only preset": {
			presets: map[string]string{"done": "<!--\nprev\n-->"},
			hookOut: map[string]string{},
			want:    map[string]string{"done": "<!--\nprev\n-->"},
		},
		"nil maps return empty": {
			presets: nil,
			hookOut: nil,
			want:    map[string]string{},
		},
		"nil preset with hook": {
			presets: nil,
			hookOut: map[string]string{"done": "<!--\nhook\n-->"},
			want:    map[string]string{"done": "<!--\nhook\n-->"},
		},
		"empty preset value replaced by hook": {
			presets: map[string]string{"done": ""},
			hookOut: map[string]string{"done": "<!--\nhook\n-->"},
			want:    map[string]string{"done": "<!--\nhook\n-->"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := mergePresets(tt.presets, tt.hookOut)
			assert.Equal(t, tt.want, got)
		})
	}
}
