package backends

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adrg/xdg"
)

type GitBackend struct {
	repoDir string
	remote  string
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

	return &GitBackend{repoDir: dir, remote: remote}, nil
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
	return filepath.Join(g.repoDir, date.Format("2006/01"), date.Format("02")+".md")
}

func (g *GitBackend) Save(content string, date time.Time) error {
	path := g.reportPath(date)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}

	if err := g.commitAndPush(path, date); err != nil {
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
		return err
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
	bytes, err := os.ReadFile(g.reportPath(date))
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (g *GitBackend) ListReports() ([]string, error) {
	var reports []string

	err := filepath.Walk(g.repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		rel, err := filepath.Rel(g.repoDir, path)
		if err != nil {
			return err
		}
		reports = append(reports, rel)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	sort.Sort(sort.Reverse(sort.StringSlice(reports)))
	return reports, nil
}

func (g *GitBackend) LoadPreviousReport(date time.Time) (time.Time, error) {
	reports, err := g.ListReports()
	if err != nil {
		return time.Time{}, err
	}

	target := normalizeReportDate(date)
	for _, rel := range reports {
		reportDate, err := parseReportDate(rel)
		if err != nil {
			continue
		}
		if !reportDate.Before(target) {
			continue
		}

		_, err = os.ReadFile(filepath.Join(g.repoDir, rel))
		if err != nil {
			return time.Time{}, err
		}
		return reportDate, nil
	}

	return time.Time{}, os.ErrNotExist
}
