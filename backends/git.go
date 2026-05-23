package backends

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/nurazon59/nippo/report"
)

type GitBackend struct {
	fs       *FilesystemBackend
	localDir string
	remote   string
	repoURL  string
}

func NewGitBackend(localDir, repoURL, remote string) (*GitBackend, error) {
	if localDir == "" {
		return nil, errors.New("git backend: local_dir is required")
	}
	if remote == "" {
		remote = "origin"
	}
	gb := &GitBackend{
		fs:       NewFilesystemBackend(localDir),
		localDir: localDir,
		remote:   remote,
		repoURL:  repoURL,
	}
	if err := gb.ensureRepo(); err != nil {
		return nil, err
	}
	return gb, nil
}

func (b *GitBackend) ensureRepo() error {
	gitDir := filepath.Join(b.localDir, ".git")
	_, err := os.Stat(gitDir)
	if err == nil {
		return nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("git backend: stat %s: %w", gitDir, err)
	}

	if err := os.MkdirAll(b.localDir, 0755); err != nil {
		return fmt.Errorf("git backend: mkdir %s: %w", b.localDir, err)
	}

	if b.repoURL != "" {
		out, err := b.runGit(b.localDir, "clone", b.repoURL, ".")
		if err != nil {
			return fmt.Errorf("git backend: clone %s: %w: %s", b.repoURL, err, out)
		}
		return nil
	}

	out, err := b.runGit(b.localDir, "init")
	if err != nil {
		return fmt.Errorf("git backend: init: %w: %s", err, out)
	}
	return nil
}

func (b *GitBackend) runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func (b *GitBackend) Save(content string, date time.Time) error {
	if err := b.fs.Save(content, date); err != nil {
		return err
	}
	if err := b.commitAndPush(date, b.fs.reportPath(date)); err != nil {
		return &PartialSaveError{
			Succeeded: []string{"filesystem"},
			Failed:    []*BackendError{{Name: "git", Err: err}},
		}
	}
	return nil
}

func (b *GitBackend) commitAndPush(date time.Time, absPath string) error {
	rel, err := filepath.Rel(b.localDir, absPath)
	if err != nil {
		return fmt.Errorf("rel path: %w", err)
	}
	rel = filepath.ToSlash(rel)

	if out, err := b.runGit(b.localDir, "add", "--", rel); err != nil {
		return fmt.Errorf("git add: %w: %s", err, out)
	}

	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = b.localDir
	if err := cmd.Run(); err == nil {
		return nil
	} else if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		return fmt.Errorf("git diff --cached: %w", err)
	}

	msg := fmt.Sprintf("report: %s", date.Format("2006-01-02"))
	if out, err := b.runGit(b.localDir, "commit", "-m", msg); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}

	if b.repoURL == "" {
		return nil
	}
	if out, err := b.runGit(b.localDir, "push", b.remote, "HEAD"); err != nil {
		return fmt.Errorf("git push: %w: %s", err, out)
	}
	return nil
}

func (b *GitBackend) LoadReport(date time.Time) (string, error) {
	return b.fs.LoadReport(date)
}

func (b *GitBackend) LoadPreviousReport(date time.Time) (time.Time, error) {
	return b.fs.LoadPreviousReport(date)
}

func (b *GitBackend) LoadLatestReport() (time.Time, error) {
	return b.fs.LoadLatestReport()
}

func (b *GitBackend) ListReports() ([]string, error) {
	return b.fs.ListReports()
}

func (b *GitBackend) Close() error {
	return b.fs.Close()
}

func (b *GitBackend) SaveReport(r *report.Report) error {
	if r == nil {
		return fmt.Errorf("git backend: report is nil")
	}
	if err := b.fs.SaveReport(r); err != nil {
		return err
	}
	if err := b.commitAndPush(r.Date, b.fs.yamlReportPath(r.Date)); err != nil {
		return &PartialSaveError{
			Succeeded: []string{"filesystem"},
			Failed:    []*BackendError{{Name: "git", Err: err}},
		}
	}
	return nil
}

func (b *GitBackend) LoadReportStruct(date time.Time) (*report.Report, error) {
	return b.fs.LoadReportStruct(date)
}

func (b *GitBackend) WriteSidecar(date time.Time, kind string, content []byte) error {
	if err := b.fs.WriteSidecar(date, kind, content); err != nil {
		return err
	}
	if err := b.commitAndPush(date, b.fs.sidecarPath(date, kind)); err != nil {
		return &PartialSaveError{
			Succeeded: []string{"filesystem"},
			Failed:    []*BackendError{{Name: "git", Err: err}},
		}
	}
	return nil
}
