package assistants

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GitRepository is a git repository in a temporary directory, isolated from
// the git configuration of whoever runs the tests.
type GitRepository struct {
	t       *testing.T
	Dir     string
	commits int
}

func NewGitRepository(t *testing.T) *GitRepository {
	t.Helper()
	repository := &GitRepository{t: t, Dir: t.TempDir()}
	repository.Git("init", "-q", "-b", "main")
	return repository
}

// Git runs git in the repository, and returns its trimmed output.
func (r *GitRepository) Git(args ...string) string {
	r.t.Helper()
	return r.GitAs("Jane Doe", "jane@example.com", args...)
}

// GitAs runs git in the repository as an author and committer.
func (r *GitRepository) GitAs(name, email string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME="+name, "GIT_AUTHOR_EMAIL="+email,
		"GIT_COMMITTER_NAME="+name, "GIT_COMMITTER_EMAIL="+email,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(r.t, err, string(out))
	return strings.TrimSpace(string(out))
}

// Commit commits a new file, with a message, as an author, and returns its
// hash. Each commit has its own file, so that branches merge without
// conflict.
func (r *GitRepository) Commit(name, email, message string) string {
	r.t.Helper()
	r.commits++
	file := fmt.Sprintf("file-%d.txt", r.commits)
	require.NoError(r.t, os.WriteFile(filepath.Join(r.Dir, file), []byte(message+"\n"), 0o600))
	r.GitAs(name, email, "add", file)
	r.GitAs(name, email, "commit", "-q", "-m", message)
	return r.Git("rev-parse", "HEAD")
}

// Every commit of the range is read, the merges included, with its author,
// its committer and its full message.
func TestGitLog(t *testing.T) {
	repository := NewGitRepository(t)
	base := repository.Commit("Jane Doe", "jane@example.com", "Initial commit")
	assisted := repository.Commit("Jane Doe", "jane@example.com",
		"Fix the login\n\nSome body.\n\nCo-Authored-By: Claude <noreply@anthropic.com>\nClaude-Session: https://claude.ai/code/session_1")
	repository.Git("checkout", "-q", "-b", "feature", base)
	copilot := repository.Commit("copilot-swe-agent[bot]", "198982749+Copilot@users.noreply.github.com", "Add a feature")
	repository.Git("checkout", "-q", "main")
	repository.Git("merge", "-q", "--no-ff", "-m", "Merge the feature", "feature")
	head := repository.Git("rev-parse", "HEAD")

	commits, err := GitLog(repository.Dir, base+".."+head)

	require.NoError(t, err)
	require.Len(t, commits, 3)
	byHash := map[string]Commit{}
	for _, c := range commits {
		byHash[c.Hash] = c
	}
	assert.Equal(t, Commit{
		Hash:           assisted,
		AuthorName:     "Jane Doe",
		AuthorEmail:    "jane@example.com",
		CommitterName:  "Jane Doe",
		CommitterEmail: "jane@example.com",
		Message:        "Fix the login\n\nSome body.\n\nCo-Authored-By: Claude <noreply@anthropic.com>\nClaude-Session: https://claude.ai/code/session_1\n",
	}, byHash[assisted])
	assert.Equal(t, "copilot-swe-agent[bot]", byHash[copilot].AuthorName)
	assert.Equal(t, "Merge the feature\n", byHash[head].Message)

	assert.Equal(t, Change{
		Assistants:      []string{ClaudeCode, Copilot},
		AssistedCommits: 2,
		TotalCommits:    3,
		SessionLinks:    []string{"https://claude.ai/code/session_1"},
	}, Summarize(commits, builtIns))
}

// An empty range is no commit at all, not an error.
func TestGitLogEmptyRange(t *testing.T) {
	repository := NewGitRepository(t)
	head := repository.Commit("Jane Doe", "jane@example.com", "Initial commit")

	commits, err := GitLog(repository.Dir, head+".."+head)

	require.NoError(t, err)
	assert.Empty(t, commits)
}

func TestGitLogUnknownRevision(t *testing.T) {
	repository := NewGitRepository(t)
	head := repository.Commit("Jane Doe", "jane@example.com", "Initial commit")

	_, err := GitLog(repository.Dir, "0000000000000000000000000000000000000000.."+head)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read the commits of 0000000000000000000000000000000000000000.."+head+": ")
}

// Only an explicit range: the previous commit is known to the CI.
func TestGitLogRangeOnly(t *testing.T) {
	for _, value := range []string{"HEAD", "..HEAD", "HEAD..", "a...b", "--all..HEAD", "a..-b"} {
		t.Run(value, func(t *testing.T) {
			_, err := GitLog(t.TempDir(), value)

			assert.EqualError(t, err, "expected a range <previous-commit>..<commit>, got "+value)
		})
	}
}

func TestGitLogWithoutGit(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := GitLog(t.TempDir(), "a..b")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "git is required to read the commits of a..b")
}
