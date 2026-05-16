package backends

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
)

type GitBackend struct {
	repoDir    string
	remote     string
	filesystem *FilesystemBackend
}

var _ ReportStorage = (*GitBackend)(nil)

func NewGitBackend(repoURL, remote string) (*GitBackend, error) {
	dir := filepath.Join(xdg.DataHome, "nippo", "git-repo")

	if repoURL != "" {
		if err := cloneOrPullRepo(dir, repoURL); err != nil {
			return nil, fmt.Errorf("failed to clone/pull repo: %w", err)
		}
		if remote == "" {
			remote = "origin"
		}
	} else {
		if err := initLocalRepo(dir); err != nil {
			return nil, fmt.Errorf("failed to init local repo: %w", err)
		}
		remote = ""
	}

	return newGitBackend(dir, remote), nil
}

func newGitBackend(repoDir, remote string) *GitBackend {
	return &GitBackend{
		repoDir:    repoDir,
		remote:     remote,
		filesystem: newFilesystemBackend(repoDir, ""),
	}
}

func cloneOrPullRepo(dir, url string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		cmd := exec.Command("git", "clone", url, dir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	cmd := exec.Command("git", "-C", dir, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func initLocalRepo(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		cmd := exec.Command("git", "init", dir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func (g *GitBackend) reportPath(date time.Time) string {
	return g.filesystem.reportPath(date)
}

func (g *GitBackend) Save(content string, date time.Time) error {
	if err := g.filesystem.Save(content, date); err != nil {
		return err
	}

	if err := g.commitAndPush(g.reportPath(date), date); err != nil {
		return fmt.Errorf("failed to commit/push: %w", err)
	}

	return nil
}

func (g *GitBackend) commitAndPush(path string, date time.Time) error {
	relPath, err := filepath.Rel(g.repoDir, path)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "-C", g.repoDir, "add", relPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	msg := fmt.Sprintf("chore: add daily report for %s", date.Format("2006-01-02"))
	cmd = exec.Command("git", "-C", g.repoDir, "commit", "-m", msg)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("git commit failed: %s: %w", strings.TrimSpace(string(out)), err)
	}

	if g.remote != "" {
		cmd = exec.Command("git", "-C", g.repoDir, "push", g.remote)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func (g *GitBackend) LoadReport(date time.Time) (string, error) {
	return g.filesystem.LoadReport(date)
}

func (g *GitBackend) ListReports() ([]string, error) {
	return g.filesystem.ListReports()
}

func (g *GitBackend) LoadPreviousReport(date time.Time) (time.Time, error) {
	return g.filesystem.LoadPreviousReport(date)
}
