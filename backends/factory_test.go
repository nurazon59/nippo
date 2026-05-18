package backends

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild_NilFallsBackToFilesystem(t *testing.T) {
	dir := t.TempDir()
	b, err := Build(nil, dir)
	require.NoError(t, err)
	defer b.Close()

	_, ok := b.(*FilesystemBackend)
	assert.True(t, ok)
}

func TestBuild_EmptyBackendsFallsBack(t *testing.T) {
	dir := t.TempDir()
	b, err := Build(&StorageConfig{}, dir)
	require.NoError(t, err)
	defer b.Close()

	_, ok := b.(*FilesystemBackend)
	assert.True(t, ok)
}

func TestBuild_SingleFilesystem(t *testing.T) {
	dir := t.TempDir()
	cfg := &StorageConfig{Backends: []BackendConfig{
		{Type: "filesystem", Filesystem: &FilesystemBackendConfig{Dir: dir}},
	}}
	b, err := Build(cfg, "/should-not-be-used")
	require.NoError(t, err)
	defer b.Close()

	fb, ok := b.(*FilesystemBackend)
	require.True(t, ok)
	assert.Equal(t, dir, fb.baseDir)
}

func TestBuild_SingleSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.db")
	cfg := &StorageConfig{Backends: []BackendConfig{
		{Type: "sqlite", SQLite: &SQLiteBackendConfig{Path: path}},
	}}
	b, err := Build(cfg, "")
	require.NoError(t, err)
	defer b.Close()

	_, ok := b.(*SQLiteBackend)
	assert.True(t, ok)
}

func TestBuild_MultiWraps(t *testing.T) {
	cfg := &StorageConfig{Backends: []BackendConfig{
		{Type: "filesystem", Filesystem: &FilesystemBackendConfig{Dir: t.TempDir()}},
		{Type: "sqlite", SQLite: &SQLiteBackendConfig{Path: filepath.Join(t.TempDir(), "r.db")}},
	}}
	b, err := Build(cfg, "")
	require.NoError(t, err)
	defer b.Close()

	_, ok := b.(*MultiBackend)
	assert.True(t, ok)
}

func TestBuild_MissingTypeErrors(t *testing.T) {
	cfg := &StorageConfig{Backends: []BackendConfig{{Type: ""}}}
	_, err := Build(cfg, "")
	require.Error(t, err)
}

func TestBuild_GitWithoutGitConfigErrors(t *testing.T) {
	cfg := &StorageConfig{Backends: []BackendConfig{{Type: "git"}}}
	_, err := Build(cfg, "")
	require.Error(t, err)
}

func TestBuild_UnknownTypeErrors(t *testing.T) {
	cfg := &StorageConfig{Backends: []BackendConfig{{Type: "rainbow"}}}
	_, err := Build(cfg, "")
	require.Error(t, err)
}
