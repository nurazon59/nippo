package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	var loadTests = map[string]struct {
		configFile string
		want       int
	}{
		"config file": {
			configFile: "testdata/config.yaml",
			want:       1,
		},
		"default when file missing": {
			configFile: filepath.Join(t.TempDir(), "missing.yaml"),
			want:       1,
		},
		"invalid yaml": {
			configFile: func() string {
				dir := t.TempDir()
				path := filepath.Join(dir, "broken.yaml")
				_ = os.WriteFile(path, []byte(": invalid"), 0o644)
				return path
			}(),
			want: 0,
		},
	}

	for name, test := range loadTests {
		t.Run(name, func(t *testing.T) {
			cfg, err := Load(test.configFile)
			if name == "invalid yaml" {
				assert.Error(t, err)
				assert.Nil(t, cfg)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.want, cfg.Version)
			if name == "config file" {
				assert.Equal(t, "done", cfg.Questions[1].ReferenceKey)
			}
		})
	}
}

func TestConfigSameDayReferenceKey(t *testing.T) {
	tests := map[string]struct {
		yaml    string
		wantKey map[int]string
	}{
		"set on second question": {
			yaml:    "version: 1\nquestions:\n  - key: done\n    label: \"やった\"\n    required: true\n  - key: todo\n    label: \"やる\"\n    required: true\n    same_day_reference_key: done\n",
			wantKey: map[int]string{0: "", 1: "done"},
		},
		"omitted on all questions": {
			yaml:    "version: 1\nquestions:\n  - key: done\n    label: \"やった\"\n  - key: todo\n    label: \"やる\"\n",
			wantKey: map[int]string{0: "", 1: ""},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(tt.yaml), 0o644))

			cfg, err := Load(path)
			require.NoError(t, err)
			for i, want := range tt.wantKey {
				assert.Equal(t, want, cfg.Questions[i].SameDayReferenceKey, "index=%d", i)
			}
		})
	}
}

func TestConfigHookTimeoutValidation(t *testing.T) {
	tests := map[string]struct {
		yaml    string
		wantErr bool
	}{
		"valid timeout": {
			yaml:    "version: 1\nhooks:\n  - name: a\n    command: \"echo ok\"\n    keys: [done]\n    timeout: 30s\n",
			wantErr: false,
		},
		"omitted timeout": {
			yaml:    "version: 1\nhooks:\n  - name: a\n    command: \"echo ok\"\n    keys: [done]\n",
			wantErr: false,
		},
		"invalid timeout rejected at load": {
			yaml:    "version: 1\nhooks:\n  - name: a\n    command: \"echo ok\"\n    keys: [done]\n    timeout: \"not-a-duration\"\n",
			wantErr: true,
		},
		"numeric without unit rejected": {
			yaml:    "version: 1\nhooks:\n  - name: a\n    command: \"echo ok\"\n    keys: [done]\n    timeout: \"30\"\n",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(tt.yaml), 0o644))
			_, err := Load(path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigQuestionType(t *testing.T) {
	tests := map[string]struct {
		yaml     string
		wantErr  bool
		wantType map[int]string
	}{
		"type 省略は text 扱い (空文字を保持)": {
			yaml:     "version: 1\nquestions:\n  - key: done\n    label: \"やった\"\n",
			wantType: map[int]string{0: ""},
		},
		"text を明示指定できる": {
			yaml:     "version: 1\nquestions:\n  - key: done\n    label: \"やった\"\n    type: text\n",
			wantType: map[int]string{0: "text"},
		},
		"task_list を明示指定できる": {
			yaml:     "version: 1\nquestions:\n  - key: done\n    label: \"やった\"\n    type: task_list\n",
			wantType: map[int]string{0: "task_list"},
		},
		"未知の type は reject": {
			yaml:    "version: 1\nquestions:\n  - key: done\n    label: \"やった\"\n    type: bogus\n",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(tt.yaml), 0o644))

			cfg, err := Load(path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			for i, want := range tt.wantType {
				assert.Equal(t, want, cfg.Questions[i].Type, "index=%d", i)
			}
		})
	}
}

func TestConfigStorage(t *testing.T) {
	tests := map[string]struct {
		yaml   func(dir string) string
		verify func(t *testing.T, dir string, cfg *Config)
	}{
		"legacy storage_dir": {
			yaml: func(dir string) string { return "version: 1\nstorage_dir: " + dir + "\n" },
			verify: func(t *testing.T, dir string, cfg *Config) {
				assert.Equal(t, dir, cfg.StorageDir)
				assert.Nil(t, cfg.Storage)
			},
		},
		"new backends list": {
			yaml: func(dir string) string {
				return "version: 1\nstorage:\n  backends:\n    - type: filesystem\n      filesystem:\n        dir: " + dir + "\n"
			},
			verify: func(t *testing.T, dir string, cfg *Config) {
				require.NotNil(t, cfg.Storage)
				require.Len(t, cfg.Storage.Backends, 1)
				assert.Equal(t, "filesystem", cfg.Storage.Backends[0].Type)
				require.NotNil(t, cfg.Storage.Backends[0].Filesystem)
				assert.Equal(t, dir, cfg.Storage.Backends[0].Filesystem.Dir)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(tt.yaml(dir)), 0o644))

			cfg, err := Load(path)
			require.NoError(t, err)
			tt.verify(t, dir, cfg)

			storage, err := NewStorage(cfg)
			require.NoError(t, err)
			defer storage.Close()
		})
	}
}
