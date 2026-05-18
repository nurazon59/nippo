package backends

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	tests := map[string]struct {
		cfg          func(t *testing.T) *StorageConfig
		fallbackDir  func(t *testing.T) string
		wantErr      bool
		assertResult func(t *testing.T, b ReportStorage)
	}{
		"nil cfg falls back to filesystem": {
			cfg:         func(t *testing.T) *StorageConfig { return nil },
			fallbackDir: func(t *testing.T) string { return t.TempDir() },
			assertResult: func(t *testing.T, b ReportStorage) {
				_, ok := b.(*FilesystemBackend)
				assert.True(t, ok)
			},
		},
		"empty backends falls back to filesystem": {
			cfg:         func(t *testing.T) *StorageConfig { return &StorageConfig{} },
			fallbackDir: func(t *testing.T) string { return t.TempDir() },
			assertResult: func(t *testing.T, b ReportStorage) {
				_, ok := b.(*FilesystemBackend)
				assert.True(t, ok)
			},
		},
		"single filesystem backend": {
			cfg: func(t *testing.T) *StorageConfig {
				return &StorageConfig{Backends: []BackendConfig{
					{Type: "filesystem", Filesystem: &FilesystemBackendConfig{Dir: t.TempDir()}},
				}}
			},
			fallbackDir: func(t *testing.T) string { return "/should-not-be-used" },
			assertResult: func(t *testing.T, b ReportStorage) {
				_, ok := b.(*FilesystemBackend)
				assert.True(t, ok)
			},
		},
		"single sqlite backend": {
			cfg: func(t *testing.T) *StorageConfig {
				return &StorageConfig{Backends: []BackendConfig{
					{Type: "sqlite", SQLite: &SQLiteBackendConfig{Path: filepath.Join(t.TempDir(), "r.db")}},
				}}
			},
			fallbackDir: func(t *testing.T) string { return "" },
			assertResult: func(t *testing.T, b ReportStorage) {
				_, ok := b.(*SQLiteBackend)
				assert.True(t, ok)
			},
		},
		"two backends wrap in MultiBackend": {
			cfg: func(t *testing.T) *StorageConfig {
				return &StorageConfig{Backends: []BackendConfig{
					{Type: "filesystem", Filesystem: &FilesystemBackendConfig{Dir: t.TempDir()}},
					{Type: "sqlite", SQLite: &SQLiteBackendConfig{Path: filepath.Join(t.TempDir(), "r.db")}},
				}}
			},
			fallbackDir: func(t *testing.T) string { return "" },
			assertResult: func(t *testing.T, b ReportStorage) {
				_, ok := b.(*MultiBackend)
				assert.True(t, ok)
			},
		},
		"missing type errors": {
			cfg: func(t *testing.T) *StorageConfig {
				return &StorageConfig{Backends: []BackendConfig{{Type: ""}}}
			},
			fallbackDir: func(t *testing.T) string { return "" },
			wantErr:     true,
		},
		"git without git sub-config errors": {
			cfg: func(t *testing.T) *StorageConfig {
				return &StorageConfig{Backends: []BackendConfig{{Type: "git"}}}
			},
			fallbackDir: func(t *testing.T) string { return "" },
			wantErr:     true,
		},
		"unknown type errors": {
			cfg: func(t *testing.T) *StorageConfig {
				return &StorageConfig{Backends: []BackendConfig{{Type: "rainbow"}}}
			},
			fallbackDir: func(t *testing.T) string { return "" },
			wantErr:     true,
		},
		"sqlite without sub-config errors": {
			cfg: func(t *testing.T) *StorageConfig {
				return &StorageConfig{Backends: []BackendConfig{{Type: "sqlite"}}}
			},
			fallbackDir: func(t *testing.T) string { return "" },
			wantErr:     true,
		},
		"filesystem without dir errors": {
			cfg: func(t *testing.T) *StorageConfig {
				return &StorageConfig{Backends: []BackendConfig{
					{Type: "filesystem", Filesystem: &FilesystemBackendConfig{}},
				}}
			},
			fallbackDir: func(t *testing.T) string { return "" },
			wantErr:     true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b, err := Build(tt.cfg(t), tt.fallbackDir(t))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			defer b.Close()
			if tt.assertResult != nil {
				tt.assertResult(t, b)
			}
		})
	}
}
