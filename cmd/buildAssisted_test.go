package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const assistedChangeSet = `{"data": {"setBuildAssistedChangeProperty": {"errors": []}}}`

var assistedArgs = []string{"--project", "my-project", "--branch", "release/1.0", "--build", "42"}

// runAssisted runs 'build assisted' with args, and returns its error, what it
// printed, and what it warned about.
func runAssisted(t *testing.T, args ...string) (err error, stdout string, stderr string) {
	t.Helper()
	cmd := cmdWithArgs(t, buildAssistedCmd, append(assistedArgs, args...)...)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	t.Cleanup(func() {
		cmd.SetOut(nil)
		cmd.SetErr(nil)
	})
	err = cmd.RunE(cmd, nil)
	return err, out.String(), errOut.String()
}

// assistedInput is what was sent to setBuildAssistedChangeProperty, by the
// last request.
func assistedInput(t *testing.T, requests *[]graphQLRequest) map[string]interface{} {
	t.Helper()
	require.NotEmpty(t, *requests)
	request := (*requests)[len(*requests)-1]
	assert.Contains(t, request.Query, "setBuildAssistedChangeProperty(input: {")
	// Set by CI, never UNKNOWN, and never from a previous build
	assert.Contains(t, request.Query, "basis: SET_BY_CI")
	assert.NotContains(t, request.Query, "previousBuildId")
	return request.Variables
}

func TestBuildAssistedExplicit(t *testing.T) {
	requests := fakeYontrackRoutes(t, map[string]string{"setBuildAssistedChangeProperty": assistedChangeSet})

	err, stdout, _ := runAssisted(t, "--assistants", "Codex,Claude Code", "--assisted-commits", "3", "--total-commits", "5",
		"--session-link", "https://claude.ai/code/session_1", "--session-link", "https://example.com/a,b")

	require.NoError(t, err)
	require.Len(t, *requests, 1)
	assert.Equal(t, map[string]interface{}{
		"project":         "my-project",
		"branch":          "release-1.0",
		"build":           "42",
		"assistants":      []interface{}{"Claude Code", "Codex"},
		"assistedCommits": float64(3),
		"totalCommits":    float64(5),
		"sessionLinks":    []interface{}{"https://claude.ai/code/session_1", "https://example.com/a,b"},
	}, assistedInput(t, requests))
	assert.Equal(t, "Build 42: 3 of 5 commits assisted (Claude Code, Codex)\n", stdout)
}

// No assisted commit is a fact too, and is set: an absent property counts as
// assisted.
func TestBuildAssistedExplicitNone(t *testing.T) {
	requests := fakeYontrackRoutes(t, map[string]string{"setBuildAssistedChangeProperty": assistedChangeSet})

	err, stdout, _ := runAssisted(t, "--total-commits", "4")

	require.NoError(t, err)
	input := assistedInput(t, requests)
	assert.Equal(t, []interface{}{}, input["assistants"])
	assert.Equal(t, float64(0), input["assistedCommits"])
	assert.Equal(t, float64(4), input["totalCommits"])
	assert.Equal(t, []interface{}{}, input["sessionLinks"])
	assert.Equal(t, "Build 42: 0 of 4 commits assisted\n", stdout)
}

func TestBuildAssistedModes(t *testing.T) {
	cases := []struct {
		args  []string
		error string
	}{
		{nil, "one of --from-git or --total-commits is required"},
		{[]string{"--from-git", "a..b", "--assistants", "Codex"},
			"--from-git is mutually exclusive with --assistants, --assisted-commits, --total-commits and --session-link"},
		{[]string{"--from-git", "a..b", "--session-link", "https://claude.ai/code/session_1"},
			"--from-git is mutually exclusive with --assistants, --assisted-commits, --total-commits and --session-link"},
		{[]string{"--assistants", "Codex", "--assisted-commits", "1"}, "--total-commits is required with --assistants, --assisted-commits and --session-link"},
	}
	for _, c := range cases {
		t.Run(strings.Join(c.args, " "), func(t *testing.T) {
			requests := fakeYontrackRoutes(t, map[string]string{})

			err, _, _ := runAssisted(t, c.args...)

			assert.EqualError(t, err, c.error)
			assert.Empty(t, *requests)
		})
	}
}

func TestBuildAssistedRefused(t *testing.T) {
	fakeYontrackRoutes(t, map[string]string{"setBuildAssistedChangeProperty": `{"data": {"setBuildAssistedChangeProperty": {"errors": [
		{"message": "The number of assisted commits (6) must not exceed the total number of commits (5)."}
	]}}}`})

	err, _, _ := runAssisted(t, "--assistants", "Codex", "--assisted-commits", "6", "--total-commits", "5")

	assert.EqualError(t, err, "1) The number of assisted commits (6) must not exceed the total number of commits (5).\n")
}

// gitRepository is a git repository in a temporary directory, isolated from
// the git configuration of whoever runs the tests, and the working directory
// of the test.
type gitRepository struct {
	t       *testing.T
	dir     string
	commits int
}

func newGitRepository(t *testing.T) *gitRepository {
	t.Helper()
	repository := &gitRepository{t: t, dir: t.TempDir()}
	repository.gitAs("Jane Doe", "jane@example.com", "init", "-q", "-b", "main")
	previous, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(repository.dir))
	t.Cleanup(func() { _ = os.Chdir(previous) })
	return repository
}

func (r *gitRepository) gitAs(name, email string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
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

// commit commits a new file as an author, and returns its hash.
func (r *gitRepository) commit(name, email, message string) string {
	r.t.Helper()
	r.commits++
	file := fmt.Sprintf("file-%d.txt", r.commits)
	require.NoError(r.t, os.WriteFile(filepath.Join(r.dir, file), []byte(message), 0o600))
	r.gitAs(name, email, "add", file)
	r.gitAs(name, email, "commit", "-q", "-m", message)
	return r.gitAs(name, email, "rev-parse", "HEAD")
}

// assistedHistory is a range of three commits: one by Claude Code, one by an
// internal agent, acme-bot, and a merge.
func assistedHistory(t *testing.T) string {
	t.Helper()
	repository := newGitRepository(t)
	base := repository.commit("Jane Doe", "jane@example.com", "Initial commit")
	repository.commit("Jane Doe", "jane@example.com",
		"Fix the login\n\nCo-Authored-By: Claude <noreply@anthropic.com>\nClaude-Session: https://claude.ai/code/session_1")
	repository.gitAs("Jane Doe", "jane@example.com", "checkout", "-q", "-b", "feature", base)
	repository.commit("acme-bot", "bot@acme.com", "Bump the dependencies")
	repository.gitAs("Jane Doe", "jane@example.com", "checkout", "-q", "main")
	repository.gitAs("Jane Doe", "jane@example.com", "merge", "-q", "--no-ff", "-m", "Merge the feature", "feature")
	head := repository.gitAs("Jane Doe", "jane@example.com", "rev-parse", "HEAD")
	return base + ".." + head
}

const agentMarkers = `{"data": {"settings": {"settingsById": {"values": {"builtInConventions": true, "patterns": [
	{"name": "Acme Bot", "type": "LOGIN", "value": "acme-bot"}
]}}}}}`

// The commits of the range are read from git, with the agent markers of
// Yontrack, every commit counting, merges included.
func TestBuildAssistedFromGit(t *testing.T) {
	gitRange := assistedHistory(t)
	requests := fakeYontrackRoutes(t, map[string]string{
		"settingsById(id: \"agent-markers\")": agentMarkers,
		"setBuildAssistedChangeProperty":      assistedChangeSet,
	})

	err, stdout, stderr := runAssisted(t, "--from-git", gitRange)

	require.NoError(t, err)
	assert.Empty(t, stderr)
	require.Len(t, *requests, 2)
	assert.Equal(t, map[string]interface{}{
		"project":         "my-project",
		"branch":          "release-1.0",
		"build":           "42",
		"assistants":      []interface{}{"Acme Bot", "Claude Code"},
		"assistedCommits": float64(2),
		"totalCommits":    float64(3),
		"sessionLinks":    []interface{}{"https://claude.ai/code/session_1"},
	}, assistedInput(t, requests))
	assert.Equal(t, "Build 42: 2 of 3 commits assisted (Acme Bot, Claude Code)\n", stdout)
}

// Reading the agent markers needs the right to read the global settings,
// which an agent or a CI token may not have: the built-in conventions still
// apply, and the custom patterns are not.
func TestBuildAssistedFromGitWithoutSettings(t *testing.T) {
	gitRange := assistedHistory(t)
	requests := fakeYontrackRoutes(t, map[string]string{
		"settingsById(id: \"agent-markers\")": `{"errors": [{"message": "Global function 'GlobalSettings' is not granted.", "extensions": {"classification": "FORBIDDEN"}}],
			"data": {"settings": {"settingsById": null}}}`,
		"setBuildAssistedChangeProperty": assistedChangeSet,
	})

	err, _, stderr := runAssisted(t, "--from-git", gitRange)

	require.NoError(t, err)
	assert.Equal(t, "Warning: this token is not allowed to read the agent markers settings of Yontrack: "+
		"only the built-in conventions are applied, not the custom patterns.\n", stderr)
	input := assistedInput(t, requests)
	assert.Equal(t, []interface{}{"Claude Code"}, input["assistants"])
	assert.Equal(t, float64(1), input["assistedCommits"])
	assert.Equal(t, float64(3), input["totalCommits"])
}

// Any other error is one.
func TestBuildAssistedFromGitSettingsError(t *testing.T) {
	gitRange := assistedHistory(t)
	requests := fakeYontrackRoutes(t, map[string]string{
		"settingsById(id: \"agent-markers\")": `{"errors": [{"message": "Something went wrong"}]}`,
		"setBuildAssistedChangeProperty":      assistedChangeSet,
	})

	err, _, _ := runAssisted(t, "--from-git", gitRange)

	assert.EqualError(t, err, "1) Something went wrong\n")
	assert.Len(t, *requests, 1)
}

// A pattern Go cannot compile, like a lookaround Java accepts, is skipped.
func TestBuildAssistedFromGitInvalidPattern(t *testing.T) {
	gitRange := assistedHistory(t)
	requests := fakeYontrackRoutes(t, map[string]string{
		"settingsById(id: \"agent-markers\")": `{"data": {"settings": {"settingsById": {"values": {"builtInConventions": true, "patterns": [
			{"name": "Lookahead", "type": "AUTHOR_EMAIL", "value": "(?=bot)bot@acme\\.com"},
			{"name": "Acme Bot", "type": "LOGIN", "value": "acme-bot"}
		]}}}}}`,
		"setBuildAssistedChangeProperty": assistedChangeSet,
	})

	err, _, stderr := runAssisted(t, "--from-git", gitRange)

	require.NoError(t, err)
	assert.Equal(t, "Warning: agent marker pattern skipped. Pattern #1 (Lookahead): invalid regular expression `(?=bot)bot@acme\\.com`\n", stderr)
	assert.Equal(t, []interface{}{"Acme Bot", "Claude Code"}, assistedInput(t, requests)["assistants"])
}

// A range git cannot resolve fails before anything is sent.
func TestBuildAssistedFromGitUnknownRange(t *testing.T) {
	assistedHistory(t)
	requests := fakeYontrackRoutes(t, map[string]string{})

	err, _, _ := runAssisted(t, "--from-git", "unknown..HEAD")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read the commits of unknown..HEAD: ")
	assert.Empty(t, *requests)
}
