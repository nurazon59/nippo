package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/nurazon59/nippo/backends"
)

// QuestionConfig の Type は "text" (空ならデフォルト) または "task_list"。
// 不明値は config load 時に reject (silent fallback 禁止)。
type QuestionConfig struct {
	Key                 string `yaml:"key"`
	Label               string `yaml:"label"`
	Required            bool   `yaml:"required"`
	Type                string `yaml:"type,omitempty"`
	ReferenceKey        string `yaml:"reference_key"`
	SameDayReferenceKey string `yaml:"same_day_reference_key"`
}

type HookConfig struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Keys    []string `yaml:"keys"`
	Timeout string   `yaml:"timeout,omitempty"`
}

type Config struct {
	Version    int                     `yaml:"version"`
	StorageDir string                  `yaml:"storage_dir"`
	Storage    *backends.StorageConfig `yaml:"storage,omitempty"`
	Questions  []QuestionConfig        `yaml:"questions"`
	Hooks      []HookConfig            `yaml:"hooks,omitempty"`
}

func Load(path string) (*Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return nil, err
	}

	err = yaml.Unmarshal(bytes, cfg)
	if err != nil {
		return nil, err
	}

	for i, h := range cfg.Hooks {
		if h.Timeout == "" {
			continue
		}
		if _, err := time.ParseDuration(h.Timeout); err != nil {
			return nil, fmt.Errorf("hooks[%d] (%s): invalid timeout %q: %w", i, h.Name, h.Timeout, err)
		}
	}

	for i, q := range cfg.Questions {
		switch q.Type {
		case "", "text", "task_list":
		default:
			return nil, fmt.Errorf("questions[%d] (%s): unsupported type %q (want \"text\" or \"task_list\")", i, q.Key, q.Type)
		}
	}
	return cfg, nil
}

func Default() *Config {
	return &Config{
		Version: 1,
		Questions: []QuestionConfig{
			{Key: "done", Label: "やった", Required: true},
			{Key: "todo", Label: "やる", Required: true},
			{Key: "thoughts", Label: "所感", Required: false},
		},
	}
}

func (c *Config) Save(path string) error {
	bytes, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0644)
}
