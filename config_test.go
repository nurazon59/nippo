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

func TestConfig_LegacyStorageDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.yaml")
	require.NoError(t, os.WriteFile(path, []byte("version: 1\nstorage_dir: /tmp/old\n"), 0o644))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/old", cfg.StorageDir)
	assert.Nil(t, cfg.Storage)

	storage, err := NewStorage(cfg)
	require.NoError(t, err)
	defer storage.Close()
}

func TestConfig_NewStorageBackends(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.yaml")
	yaml := "version: 1\nstorage:\n  backends:\n    - type: filesystem\n      filesystem:\n        dir: " + dir + "\n"
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o644))

	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg.Storage)
	require.Len(t, cfg.Storage.Backends, 1)
	assert.Equal(t, "filesystem", cfg.Storage.Backends[0].Type)
	require.NotNil(t, cfg.Storage.Backends[0].Filesystem)
	assert.Equal(t, dir, cfg.Storage.Backends[0].Filesystem.Dir)

	storage, err := NewStorage(cfg)
	require.NoError(t, err)
	defer storage.Close()
}
