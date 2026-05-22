package backends

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nurazon59/nippo/report"
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

	for _, args := range [][]string{
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
	}

	t.Cleanup(func() { _ = b.Close() })
	return b, dir
}

func TestGitBackend_Save(t *testing.T) {
	tests := map[string]struct {
		saves     [][2]string
		wantCount string
		wantLog   string
	}{
		"creates commit on new content": {
			saves:     [][2]string{{"2024-06-15", "# hi"}},
			wantCount: "1\n",
			wantLog:   "2024-06-15",
		},
		"idempotent on same content": {
			saves: [][2]string{
				{"2024-06-15", "# hi"},
				{"2024-06-15", "# hi"},
			},
			wantCount: "1\n",
			wantLog:   "2024-06-15",
		},
		"new commit on changed content": {
			saves: [][2]string{
				{"2024-06-15", "first"},
				{"2024-06-15", "second"},
			},
			wantCount: "2\n",
			wantLog:   "2024-06-15",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b, dir := newTestGitBackend(t)
			for _, s := range tt.saves {
				require.NoError(t, b.Save(s[1], mustDate(t, s[0])))
			}

			cmd := exec.Command("git", "rev-list", "--count", "HEAD")
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, string(out))

			cmd = exec.Command("git", "log", "--oneline")
			cmd.Dir = dir
			out, err = cmd.CombinedOutput()
			require.NoError(t, err)
			assert.True(t, strings.Contains(string(out), tt.wantLog), "log should mention %s, got:\n%s", tt.wantLog, out)
		})
	}
}

func TestGitBackend_LoadReport(t *testing.T) {
	tests := map[string]struct {
		setup   func(b *GitBackend, t *testing.T)
		date    string
		wantErr bool
		wantIs  error
		want    string
	}{
		"hit": {
			setup: func(b *GitBackend, t *testing.T) {
				require.NoError(t, b.Save("# content", mustDate(t, "2024-06-15")))
			},
			date: "2024-06-15",
			want: "# content",
		},
		"miss": {
			setup:   func(b *GitBackend, t *testing.T) {},
			date:    "2024-06-15",
			wantErr: true,
			wantIs:  fs.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b, _ := newTestGitBackend(t)
			tt.setup(b, t)

			got, err := b.LoadReport(mustDate(t, tt.date))
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantIs != nil {
					assert.True(t, errors.Is(err, tt.wantIs))
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitBackend_ListReports(t *testing.T) {
	tests := map[string]struct {
		saves []string
		want  []string
	}{
		"sorted desc": {
			saves: []string{"2024-06-15", "2024-06-14"},
			want: []string{
				filepath.Join("2024", "06", "15.md"),
				filepath.Join("2024", "06", "14.md"),
			},
		},
		"empty": {
			saves: nil,
			want:  nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b, _ := newTestGitBackend(t)
			for _, d := range tt.saves {
				require.NoError(t, b.Save("x", mustDate(t, d)))
			}

			got, err := b.ListReports()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitBackend_SaveReport(t *testing.T) {
	tests := map[string]struct {
		date      string
		wantRel   string
		wantCount string
	}{
		"creates yaml commit": {
			date:      "2024-06-15",
			wantRel:   filepath.Join("nippo", "2024", "06", "15.yaml"),
			wantCount: "1\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b, dir := newTestGitBackend(t)
			require.NoError(t, b.SaveReport(sampleReport(t, tt.date)))

			// .yaml が git に commit されていること (working tree から消えても残るので追跡確認)
			cmd := exec.Command("git", "ls-files", tt.wantRel)
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			require.NoError(t, err)
			assert.True(t, strings.Contains(string(out), tt.wantRel), "ls-files should contain %s, got:\n%s", tt.wantRel, out)

			cmd = exec.Command("git", "rev-list", "--count", "HEAD")
			cmd.Dir = dir
			out, err = cmd.CombinedOutput()
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, string(out))

			got, err := b.LoadReportStruct(mustDate(t, tt.date))
			require.NoError(t, err)
			assert.Equal(t, report.SupportedSchemaVersion, got.SchemaVersion)
		})
	}
}

func TestGitBackend_WriteSidecar(t *testing.T) {
	tests := map[string]struct {
		date    string
		kind    string
		content string
		wantRel string
	}{
		"markdown sidecar gets committed": {
			date:    "2024-06-15",
			kind:    ".md",
			content: "# nippo",
			wantRel: filepath.Join("nippo", "2024", "06", "15.md"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b, dir := newTestGitBackend(t)
			require.NoError(t, b.WriteSidecar(mustDate(t, tt.date), tt.kind, []byte(tt.content)))

			// working tree のファイルが書き出されていること
			got, err := os.ReadFile(filepath.Join(dir, tt.wantRel))
			require.NoError(t, err)
			assert.Equal(t, tt.content, string(got))

			// git にも追跡されていること
			cmd := exec.Command("git", "ls-files", tt.wantRel)
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			require.NoError(t, err)
			assert.True(t, strings.Contains(string(out), tt.wantRel))
		})
	}
}

func TestGitBackend_NewValidation(t *testing.T) {
	skipIfNoGit(t)
	tests := map[string]struct {
		localDir func(t *testing.T) string
		wantErr  bool
	}{
		"empty local_dir errors": {
			localDir: func(t *testing.T) string { return "" },
			wantErr:  true,
		},
		"valid local_dir initializes repo": {
			localDir: func(t *testing.T) string { return t.TempDir() },
			wantErr:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b, err := NewGitBackend(tt.localDir(t), "", "")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			defer b.Close()
		})
	}
}
