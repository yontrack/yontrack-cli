package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	config "yontrack/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const slotGetResponse = `{"data": {"environmentByName": {"slots": [{"id": "slot-1", "lastDeployedPipeline": null}]}}}`

// --agent-session and --agent-session-link are on every command, and every
// request carries them.
func TestAgentSessionFromFlags(t *testing.T) {
	t.Setenv("YONTRACK_AGENT_SESSION", "")
	t.Setenv("YONTRACK_AGENT_SESSION_LINK", "")
	requests := fakeYontrackRoutes(t, map[string]string{"environmentByName": slotGetResponse})
	cmd := cmdWithArgs(t, slotGetCmd, "--project", "my-project", "--environment", "production",
		"--agent-session", "session-1", "--agent-session-link", "https://claude.ai/code/session_1")

	require.NoError(t, cmd.RunE(cmd, nil))

	require.Len(t, *requests, 1)
	assert.Equal(t, "session-1", (*requests)[0].Header.Get("X-Yontrack-Agent-Session"))
	assert.Equal(t, "https://claude.ai/code/session_1", (*requests)[0].Header.Get("X-Yontrack-Agent-Session-Link"))
}

func TestAgentSessionFromEnvironment(t *testing.T) {
	t.Setenv("YONTRACK_AGENT_SESSION", "session-2")
	t.Setenv("YONTRACK_AGENT_SESSION_LINK", "https://claude.ai/code/session_2")
	requests := fakeYontrackRoutes(t, map[string]string{"environmentByName": slotGetResponse})
	cmd := cmdWithArgs(t, slotGetCmd, "--project", "my-project", "--environment", "production")

	require.NoError(t, cmd.RunE(cmd, nil))

	require.Len(t, *requests, 1)
	assert.Equal(t, "session-2", (*requests)[0].Header.Get("X-Yontrack-Agent-Session"))
	assert.Equal(t, "https://claude.ai/code/session_2", (*requests)[0].Header.Get("X-Yontrack-Agent-Session-Link"))
}

// runCLI runs the CLI with args in a process of its own, as a pipeline does,
// against the configuration of the test, and returns its exit code and what
// it printed.
func runCLI(t *testing.T, args ...string) (code int, stdout string, stderr string) {
	t.Helper()
	args = append([]string{"--config", config.ConfigFilePath}, args...)
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	process := exec.Command(os.Args[0], "-test.run=^TestCLIProcess$")
	process.Env = append(os.Environ(), "YONTRACK_TEST_CLI_ARGS="+string(encoded))
	var out, errOut bytes.Buffer
	process.Stdout = &out
	process.Stderr = &errOut
	err = process.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), out.String(), errOut.String()
	}
	require.NoError(t, err)
	return 0, out.String(), errOut.String()
}

// A failing command prints its error once and exits with 1. Cobra prints the
// usage after the error, but for an unknown command, which gets a hint to
// --help instead.
func TestCLIErrors(t *testing.T) {
	fakeYontrackRoutes(t, map[string]string{"readiness(": `{"errors": [{"message": "Promotion level not found: GOLD"}]}`})
	cases := []struct {
		name  string
		args  []string
		err   string
		usage bool
	}{
		{"command error", append([]string{"build", "readiness", "--promotion", "GOLD"}, readinessArgs...), "Promotion level not found: GOLD", true},
		{"unknown flag", []string{"build", "readiness", "--unknown"}, "Error: unknown flag: --unknown", true},
		{"unknown command", []string{"unknown"}, `Error: unknown command "unknown" for "yontrack"`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, _, stderr := runCLI(t, c.args...)

			assert.Equal(t, 1, code)
			assert.Equal(t, 1, strings.Count(stderr, "Error:"), "the error is printed once: %s", stderr)
			assert.Contains(t, stderr, c.err)
			assert.Equal(t, c.usage, strings.Contains(stderr, "Usage:"), "usage: %s", stderr)
		})
	}
}

// TestCLIProcess is the process runCLI starts: it runs the CLI, and nothing
// else.
func TestCLIProcess(t *testing.T) {
	encoded := os.Getenv("YONTRACK_TEST_CLI_ARGS")
	if encoded == "" {
		t.Skip("run by runCLI only")
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(encoded), &args))
	rootCmd.SetArgs(args)
	Execute()
	os.Exit(0)
}
