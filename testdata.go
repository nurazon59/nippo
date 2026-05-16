package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type Fixture struct {
	t       *testing.T
	storage *Storage
}

func New(t *testing.T) *Fixture {
	return &Fixture{t: t}
}

func (f *Fixture) NewStorage() *Storage {
	if f.storage == nil {
		s, err := NewStorage(f.t.TempDir())
		require.NoError(f.t, err)
		f.storage = s
	}
	return f.storage
}

func (f *Fixture) Save(date string, content string) {
	t, err := time.Parse("2006-01-02", date)
	require.NoError(f.t, err)
	require.NoError(f.t, f.NewStorage().Save(content, t))
}

func (f *Fixture) LoadConfig(path string) *Config {
	cfg, err := Load(path)
	require.NoError(f.t, err)
	return cfg
}

func (f *Fixture) DefaultConfig() *Config {
	return Default()
}
