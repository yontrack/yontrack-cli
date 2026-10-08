package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const readinessReady = `{"data": {"builds": [{"readiness": {"ready": true, "missing": []}}]}}`

const readinessNotReady = `{"data": {"builds": [{"readiness": {"ready": false, "missing": [
	{"kind": "VALIDATION", "name": "tests", "message": "The tests validation has not passed."},
	{"kind": "MANUAL", "name": "GOLD", "message": "GOLD is granted by a person."}
]}}]}}`

// runWithOutput runs a command with args, and returns its error and what it
// printed.
func runWithOutput(t *testing.T, cmd *cobra.Command, args ...string) (error, string) {
	t.Helper()
	cmdWithArgs(t, cmd, args...)
	var out bytes.Buffer
	cmd.SetOut(&out)
	t.Cleanup(func() { cmd.SetOut(nil) })
	err := cmd.RunE(cmd, nil)
	return err, out.String()
}

var readinessArgs = []string{"--project", "my-project", "--branch", "release/1.0", "--build", "42"}

func TestBuildReadinessReady(t *testing.T) {
	requests := fakeYontrackRoutes(t, map[string]string{"readiness(": readinessReady})

	err, out := runWithOutput(t, buildReadinessCmd, append(readinessArgs, "--promotion", "GOLD")...)

	require.NoError(t, err)
	assert.Equal(t, "ready\n", out)
	require.Len(t, *requests, 1)
	assert.Equal(t, map[string]interface{}{
		"project":        "my-project",
		"branch":         "release-1.0",
		"build":          "42",
		"promotionLevel": "GOLD",
	}, (*requests)[0].Variables)
}

// Not ready is not an error, but a pipeline must be able to tell it from a
// ready build: the exit code is 2.
func TestBuildReadinessNotReady(t *testing.T) {
	requests := fakeYontrackRoutes(t, map[string]string{"readiness(": readinessNotReady})

	err, out := runWithOutput(t, buildReadinessCmd, append(readinessArgs, "--slot", "slot-1")...)

	var exit *exitCodeError
	require.True(t, errors.As(err, &exit), "expected an exit code, got %v", err)
	assert.Equal(t, 2, exit.code)
	assert.Equal(t, "not ready\n"+
		"  - VALIDATION tests: The tests validation has not passed.\n"+
		"  - MANUAL GOLD: GOLD is granted by a person.\n", out)
	assert.Equal(t, "slot-1", (*requests)[0].Variables["slotId"])
	assert.NotContains(t, (*requests)[0].Variables, "promotionLevel")
}

// The JSON output is the readiness as Yontrack returns it, and does not change
// the exit code.
func TestBuildReadinessJSON(t *testing.T) {
	fakeYontrackRoutes(t, map[string]string{"readiness(": readinessNotReady})

	err, out := runWithOutput(t, buildReadinessCmd, append(readinessArgs, "--promotion", "GOLD", "--output", "json")...)

	var exit *exitCodeError
	require.True(t, errors.As(err, &exit))
	assert.Equal(t, 2, exit.code)
	assert.JSONEq(t, `{"ready": false, "missing": [
		{"kind": "VALIDATION", "name": "tests", "message": "The tests validation has not passed."},
		{"kind": "MANUAL", "name": "GOLD", "message": "GOLD is granted by a person."}
	]}`, out)
}

func TestBuildReadinessTarget(t *testing.T) {
	fakeYontrackRoutes(t, map[string]string{"readiness(": readinessReady})

	err, _ := runWithOutput(t, buildReadinessCmd, readinessArgs...)
	assert.EqualError(t, err, "one of --promotion or --slot is required")

	err, _ = runWithOutput(t, buildReadinessCmd, append(readinessArgs, "--promotion", "GOLD", "--slot", "slot-1")...)
	assert.EqualError(t, err, "--promotion and --slot are mutually exclusive")
}

func TestBuildReadinessBuildNotFound(t *testing.T) {
	fakeYontrackRoutes(t, map[string]string{"readiness(": `{"data": {"builds": []}}`})

	err, _ := runWithOutput(t, buildReadinessCmd, append(readinessArgs, "--promotion", "GOLD")...)

	assert.EqualError(t, err, "no build 42 in branch release-1.0 of project my-project")
}

func TestBuildReadinessUnknownOutput(t *testing.T) {
	fakeYontrackRoutes(t, map[string]string{"readiness(": readinessReady})

	err, _ := runWithOutput(t, buildReadinessCmd, append(readinessArgs, "--promotion", "GOLD", "--output", "env")...)

	assert.EqualError(t, err, "unknown output format: env")
}

// The exit codes, as a pipeline sees them: the CLI runs in a process of its
// own.
func TestBuildReadinessExitCodes(t *testing.T) {
	cases := []struct {
		name     string
		response string
		code     int
		stdout   string
		stderr   string
	}{
		{"ready", readinessReady, 0, "ready\n", ""},
		{"not ready", readinessNotReady, 2, "not ready\n", ""},
		{"error", `{"errors": [{"message": "Promotion level not found: GOLD"}]}`, 1, "", "Promotion level not found: GOLD"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fakeYontrackRoutes(t, map[string]string{"readiness(": c.response})

			code, stdout, stderr := runCLI(t, append([]string{"build", "readiness", "--promotion", "GOLD"}, readinessArgs...)...)

			assert.Equal(t, c.code, code)
			assert.Contains(t, stdout, c.stdout)
			if c.stderr == "" {
				// Neither an error nor the usage: the output tells why
				assert.Empty(t, stderr)
			} else {
				assert.Equal(t, 1, strings.Count(stderr, c.stderr), "the error is printed once: %s", stderr)
			}
		})
	}
}
