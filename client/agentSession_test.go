package client

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	config "yontrack/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeServer records the headers of every request it receives, and answers
// them with an empty GraphQL result, or 201 for an evidence.
func fakeServer(t *testing.T) (*config.Config, *[]http.Header) {
	t.Helper()
	var headers []http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = append(headers, r.Header.Clone())
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/graphql" {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = w.Write([]byte(`{"data": {}}`))
	}))
	t.Cleanup(server.Close)
	return &config.Config{URL: server.URL, Token: "token"}, &headers
}

// withAgentSession sets the session as --agent-session and
// --agent-session-link do, and clears the environment.
func withAgentSession(t *testing.T, session, link string) {
	t.Helper()
	t.Setenv("YONTRACK_AGENT_SESSION", "")
	t.Setenv("YONTRACK_AGENT_SESSION_LINK", "")
	previousSession, previousLink := config.AgentSession, config.AgentSessionLink
	config.AgentSession, config.AgentSessionLink = session, link
	t.Cleanup(func() { config.AgentSession, config.AgentSessionLink = previousSession, previousLink })
}

// Every request carries the session, the GraphQL calls and the uploads of
// evidence alike, whatever the token: Yontrack decides whether to read it.
func TestAgentSessionHeadersOnEveryRequest(t *testing.T) {
	withAgentSession(t, "session-1", "https://claude.ai/code/session_1")
	cfg, headers := fakeServer(t)
	evidence := filepath.Join(t.TempDir(), "report.txt")
	require.NoError(t, os.WriteFile(evidence, []byte("report"), 0o600))

	require.NoError(t, GraphQLCall(cfg, `{ projects { id } }`, nil, &struct{}{}))
	require.NoError(t, UploadEvidence(cfg, "1", evidence, EvidenceSource{}))

	require.Len(t, *headers, 2)
	for _, header := range *headers {
		assert.Equal(t, "session-1", header.Get("X-Yontrack-Agent-Session"))
		assert.Equal(t, "https://claude.ai/code/session_1", header.Get("X-Yontrack-Agent-Session-Link"))
		assert.Equal(t, "token", header.Get("X-Ontrack-Token"))
	}
}

func TestAgentSessionHeadersFromEnvironment(t *testing.T) {
	withAgentSession(t, "", "")
	t.Setenv("YONTRACK_AGENT_SESSION", "session-2")
	t.Setenv("YONTRACK_AGENT_SESSION_LINK", "https://example.com/session/2")
	cfg, headers := fakeServer(t)

	require.NoError(t, GraphQLCall(cfg, `{ projects { id } }`, nil, &struct{}{}))

	assert.Equal(t, "session-2", (*headers)[0].Get("X-Yontrack-Agent-Session"))
	assert.Equal(t, "https://example.com/session/2", (*headers)[0].Get("X-Yontrack-Agent-Session-Link"))
}

// The flags win over the environment.
func TestAgentSessionFlagsWinOverEnvironment(t *testing.T) {
	withAgentSession(t, "from-flag", "https://example.com/flag")
	t.Setenv("YONTRACK_AGENT_SESSION", "from-env")
	t.Setenv("YONTRACK_AGENT_SESSION_LINK", "https://example.com/env")
	cfg, headers := fakeServer(t)

	require.NoError(t, GraphQLCall(cfg, `{ projects { id } }`, nil, &struct{}{}))

	assert.Equal(t, "from-flag", (*headers)[0].Get("X-Yontrack-Agent-Session"))
	assert.Equal(t, "https://example.com/flag", (*headers)[0].Get("X-Yontrack-Agent-Session-Link"))
}

// Without a session, no header at all rather than empty ones.
func TestNoAgentSessionNoHeaders(t *testing.T) {
	withAgentSession(t, "", "")
	cfg, headers := fakeServer(t)

	require.NoError(t, GraphQLCall(cfg, `{ projects { id } }`, nil, &struct{}{}))

	assert.NotContains(t, (*headers)[0], "X-Yontrack-Agent-Session")
	assert.NotContains(t, (*headers)[0], "X-Yontrack-Agent-Session-Link")
}

// A link without a session is still sent: each header stands on its own.
func TestAgentSessionLinkAlone(t *testing.T) {
	withAgentSession(t, "", "https://example.com/session/3")
	cfg, headers := fakeServer(t)

	require.NoError(t, GraphQLCall(cfg, `{ projects { id } }`, nil, &struct{}{}))

	assert.NotContains(t, (*headers)[0], "X-Yontrack-Agent-Session")
	assert.Equal(t, "https://example.com/session/3", (*headers)[0].Get("X-Yontrack-Agent-Session-Link"))
}
