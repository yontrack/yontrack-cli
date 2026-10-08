package cmd

import (
	"testing"

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
