package backends

import (
	"fmt"
)

type StorageConfig struct {
	Backends []BackendConfig `yaml:"backends"`
}

type BackendConfig struct {
	Type       string                   `yaml:"type"`
	Filesystem *FilesystemBackendConfig `yaml:"filesystem,omitempty"`
	Git        *GitBackendConfig        `yaml:"git,omitempty"`
	SQLite     *SQLiteBackendConfig     `yaml:"sqlite,omitempty"`
}

type FilesystemBackendConfig struct {
	Dir string `yaml:"dir"`
}

type GitBackendConfig struct {
	RepoURL  string `yaml:"repo_url"`
	Remote   string `yaml:"remote,omitempty"`
	LocalDir string `yaml:"local_dir,omitempty"`
}

type SQLiteBackendConfig struct {
	Path string `yaml:"path"`
}

func Build(cfg *StorageConfig, fallbackDir string) (ReportStorage, error) {
	if cfg == nil || len(cfg.Backends) == 0 {
		return NewFilesystemBackend(fallbackDir), nil
	}

	built := make([]NamedBackend, 0, len(cfg.Backends))
	for i, bc := range cfg.Backends {
		b, err := buildOne(bc)
		if err != nil {
			return nil, fmt.Errorf("backends[%d]: %w", i, err)
		}
		built = append(built, NamedBackend{Name: bc.Type, Backend: b})
	}

	if len(built) == 1 {
		return built[0].Backend, nil
	}
	return NewMultiBackend(built), nil
}

func buildOne(bc BackendConfig) (ReportStorage, error) {
	switch bc.Type {
	case "filesystem":
		if bc.Filesystem == nil {
			return nil, fmt.Errorf("type=filesystem requires filesystem config")
		}
		if bc.Filesystem.Dir == "" {
			return nil, fmt.Errorf("filesystem.dir is required")
		}
		return NewFilesystemBackend(bc.Filesystem.Dir), nil
	case "git":
		if bc.Git == nil {
			return nil, fmt.Errorf("type=git requires git config")
		}
		return NewGitBackend(bc.Git.LocalDir, bc.Git.RepoURL, bc.Git.Remote)
	case "sqlite":
		if bc.SQLite == nil {
			return nil, fmt.Errorf("type=sqlite requires sqlite config")
		}
		return NewSQLiteBackend(bc.SQLite.Path)
	case "":
		return nil, fmt.Errorf("type is required")
	default:
		return nil, fmt.Errorf("unknown backend type: %q", bc.Type)
	}
}
