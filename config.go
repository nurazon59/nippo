package main

import (
	"errors"
	"io/fs"
	"os"

	"github.com/goccy/go-yaml"
)

type QuestionConfig struct {
	Key      string `yaml:"key"`
	Label    string `yaml:"label"`
	Required bool   `yaml:"required"`
}

type Config struct {
	Version    int              `yaml:"version"`
	StorageDir string           `yaml:"storage_dir"`
	Questions  []QuestionConfig `yaml:"questions"`
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
