package backends

import (
	"errors"
	"io/fs"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func newTestGitBackend(t *testing.T) (*GitBackend, string) {
	t.Helper()
	skipIfNoGit(t)
	dir := t.TempDir()
	b, err := NewGitBackend(dir, "", "")
	require.NoError(t, err)

	cmd := exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "config", "user.name", "test")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	t.Cleanup(func() { _ = b.Close() })
	return b, dir
}

func TestGitBackend_SaveCreatesCommit(t *testing.T) {
	b, dir := newTestGitBackend(t)
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	require.NoError(t, b.Save("# hi", date))

	cmd := exec.Command("git", "log", "--oneline")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "2024-06-15")
}

func TestGitBackend_SaveIdempotentSameContent(t *testing.T) {
	b, dir := newTestGitBackend(t)
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	require.NoError(t, b.Save("# hi", date))
	require.NoError(t, b.Save("# hi", date))

	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Equal(t, "1\n", string(out))
}

func TestGitBackend_LoadReport(t *testing.T) {
	b, _ := newTestGitBackend(t)
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	require.NoError(t, b.Save("# content", date))
	got, err := b.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, "# content", got)
}

func TestGitBackend_LoadReportNotFound(t *testing.T) {
	b, _ := newTestGitBackend(t)
	_, err := b.LoadReport(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestGitBackend_ListReports(t *testing.T) {
	b, _ := newTestGitBackend(t)
	for _, d := range []string{"2024-06-15", "2024-06-14"} {
		date, err := time.Parse("2006-01-02", d)
		require.NoError(t, err)
		require.NoError(t, b.Save("x", date))
	}

	got, err := b.ListReports()
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("2024", "06", "15.md"),
		filepath.Join("2024", "06", "14.md"),
	}, got)
}

func TestGitBackend_NewRequiresLocalDir(t *testing.T) {
	_, err := NewGitBackend("", "", "")
	require.Error(t, err)
}
