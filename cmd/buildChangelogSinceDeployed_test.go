package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sinceDeployedBoundariesResponse = `{"data": {
	"builds": [{"id": "11410"}],
	"slotById": {"lastDeployedPipeline": {"build": {"id": "11407"}}}
}}`

const sinceDeployedChangeLogResponse = `{"data": {"scmChangeLog": {"commits": [
	{"commit": {"id": "a1b2c3d4e5", "shortId": "a1b2c3d", "message": "Fix the login\n\nCo-Authored-By: Claude <noreply@anthropic.com>\nClaude-Session: https://claude.ai/code/session_1", "author": "Jane Doe",
		"assistants": [{"name": "Claude Code", "markers": ["CO_AUTHOR", "SESSION_TRAILER"], "sessionLink": "https://claude.ai/code/session_1"}]}},
	{"commit": {"id": "f6e5d4c3b2", "shortId": "f6e5d4c", "message": "Bump the version", "author": "John Doe", "assistants": []}}
]}}}`

var sinceDeployedArgs = []string{"--project", "my-project", "--branch", "main", "--build", "45", "--slot", "slot-1"}

// The change log goes from the build the slot last deployed to the candidate.
func TestSinceDeployed(t *testing.T) {
	requests := fakeYontrackRoutes(t, map[string]string{
		"slotById(":     sinceDeployedBoundariesResponse,
		"scmChangeLog(": sinceDeployedChangeLogResponse,
	})

	err, out := runWithOutput(t, buildChangelogSinceDeployedCmd, sinceDeployedArgs...)

	require.NoError(t, err)
	assert.Equal(t,
		"a1b2c3d Fix the login (Jane Doe) [Claude Code: CO_AUTHOR, SESSION_TRAILER]\n"+
			"f6e5d4c Bump the version (John Doe)\n", out)
	require.Len(t, *requests, 2)
	assert.Equal(t, map[string]interface{}{
		"project": "my-project",
		"branch":  "main",
		"build":   "45",
		"slot":    "slot-1",
	}, (*requests)[0].Variables)
	assert.Contains(t, (*requests)[0].Query, "lastDeployedPipeline")
	assert.Equal(t, map[string]interface{}{"from": float64(11407), "to": float64(11410)}, (*requests)[1].Variables)
	assert.Contains(t, (*requests)[1].Query, "assistants {")
}

func TestSinceDeployedJSON(t *testing.T) {
	fakeYontrackRoutes(t, map[string]string{
		"slotById(":     sinceDeployedBoundariesResponse,
		"scmChangeLog(": sinceDeployedChangeLogResponse,
	})

	err, out := runWithOutput(t, buildChangelogSinceDeployedCmd, append(sinceDeployedArgs, "--output", "json")...)

	require.NoError(t, err)
	assert.JSONEq(t, `{"commits": [
		{"commit": {"id": "a1b2c3d4e5", "shortId": "a1b2c3d", "message": "Fix the login\n\nCo-Authored-By: Claude <noreply@anthropic.com>\nClaude-Session: https://claude.ai/code/session_1", "author": "Jane Doe",
			"assistants": [{"name": "Claude Code", "markers": ["CO_AUTHOR", "SESSION_TRAILER"], "sessionLink": "https://claude.ai/code/session_1"}]}},
		{"commit": {"id": "f6e5d4c3b2", "shortId": "f6e5d4c", "message": "Bump the version", "author": "John Doe", "assistants": []}}
	]}`, out)
}

// A slot which never completed a deployment has nothing to compare with.
func TestSinceDeployedNothingDeployed(t *testing.T) {
	requests := fakeYontrackRoutes(t, map[string]string{
		"slotById(": `{"data": {"builds": [{"id": "11410"}], "slotById": {"lastDeployedPipeline": null}}}`,
	})

	err, _ := runWithOutput(t, buildChangelogSinceDeployedCmd, sinceDeployedArgs...)

	assert.EqualError(t, err, "nothing deployed in slot slot-1 yet")
	assert.Len(t, *requests, 1)
}

func TestSinceDeployedNoSlot(t *testing.T) {
	fakeYontrackRoutes(t, map[string]string{
		"slotById(": `{"data": {"builds": [{"id": "11410"}], "slotById": null}}`,
	})

	err, _ := runWithOutput(t, buildChangelogSinceDeployedCmd, sinceDeployedArgs...)

	assert.EqualError(t, err, "no slot with ID slot-1")
}

func TestSinceDeployedNoBuild(t *testing.T) {
	fakeYontrackRoutes(t, map[string]string{
		"slotById(": `{"data": {"builds": [], "slotById": {"lastDeployedPipeline": {"build": {"id": "11407"}}}}}`,
	})

	err, _ := runWithOutput(t, buildChangelogSinceDeployedCmd, sinceDeployedArgs...)

	assert.EqualError(t, err, "no build 45 in branch main of project my-project")
}

// The slot is required: there is nothing to compare with otherwise.
func TestSinceDeployedSlotRequired(t *testing.T) {
	fakeYontrackRoutes(t, map[string]string{})

	err, _ := runWithOutput(t, buildChangelogSinceDeployedCmd, "--project", "my-project", "--branch", "main", "--build", "45")

	assert.EqualError(t, err, "--slot is required")
}
