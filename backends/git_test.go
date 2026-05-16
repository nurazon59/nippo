package backends

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGitBackend(t *testing.T) *GitBackend {
	t.Helper()
	dir := t.TempDir()

	cmd := exec.Command("git", "init", dir)
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "-C", dir, "config", "user.email", "test@example.com")
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", dir, "config", "user.name", "Test User")
	require.NoError(t, cmd.Run())

	return &GitBackend{repoDir: dir, remote: ""}
}

func TestGitBackend_SaveAndLoad(t *testing.T) {
	backend := setupGitBackend(t)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	content := "# test report"

	err := backend.Save(content, date)
	require.NoError(t, err)

	loaded, err := backend.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, content, loaded)
}

func TestGitBackend_SaveIdempotent(t *testing.T) {
	backend := setupGitBackend(t)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	content := "# test report"

	err := backend.Save(content, date)
	require.NoError(t, err)

	err = backend.Save(content, date)
	require.NoError(t, err)
}

func TestGitBackend_LoadNotFound(t *testing.T) {
	backend := setupGitBackend(t)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	_, err := backend.LoadReport(date)
	require.Error(t, err)
}

func TestGitBackend_ListReports(t *testing.T) {
	backend := setupGitBackend(t)

	dates := []time.Time{
		time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 5, 30, 0, 0, 0, 0, time.UTC),
	}

	for _, d := range dates {
		err := backend.Save("# report", d)
		require.NoError(t, err)
	}

	reports, err := backend.ListReports()
	require.NoError(t, err)
	assert.Len(t, reports, 3)
	assert.Equal(t, "2024/06/15.md", reports[0])
}

func TestGitBackend_ListReportsIgnoresNonReportMarkdown(t *testing.T) {
	backend := setupGitBackend(t)

	err := os.WriteFile(filepath.Join(backend.repoDir, "README.md"), []byte("# readme"), 0644)
	require.NoError(t, err)
	err = backend.Save("# report", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	reports, err := backend.ListReports()
	require.NoError(t, err)
	assert.Equal(t, []string{"2024/06/15.md"}, reports)
}

func TestGitBackend_LoadPreviousReport(t *testing.T) {
	backend := setupGitBackend(t)

	date1 := time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	err := backend.Save("# report 14", date1)
	require.NoError(t, err)
	err = backend.Save("# report 15", date2)
	require.NoError(t, err)

	previous, err := backend.LoadPreviousReport(time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, date2, previous)
}

func TestGitBackend_LoadPreviousReport_QueryMatchesDate(t *testing.T) {
	backend := setupGitBackend(t)

	date1 := time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	err := backend.Save("# report 14", date1)
	require.NoError(t, err)
	err = backend.Save("# report 15", date2)
	require.NoError(t, err)

	previous, err := backend.LoadPreviousReport(date2)
	require.NoError(t, err)
	assert.Equal(t, date1, previous)
}

func TestGitBackend_NoRemoteForLocal(t *testing.T) {
	backend := setupGitBackend(t)

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	err := backend.Save("# report", date)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(backend.repoDir, ".git"))
	require.NoError(t, err)
}
