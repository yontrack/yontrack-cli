package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	config "yontrack/config"

	"github.com/stretchr/testify/assert"
)

func answering(t *testing.T, response string) *config.Config {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	return &config.Config{URL: server.URL}
}

// A call Yontrack refuses for lack of rights can be told from any other error,
// for a command to do without what it was not allowed to read.
func TestGraphQLCallForbidden(t *testing.T) {
	cfg := answering(t, `{"errors": [{"message": "Global function 'GlobalSettings' is not granted.", "extensions": {"classification": "FORBIDDEN"}}], "data": {"settings": {"settingsById": null}}}`)

	err := GraphQLCall(cfg, `{ settings { settingsById(id: "agent-markers") { values } } }`, nil, &struct{}{})

	assert.EqualError(t, err, "1) Global function 'GlobalSettings' is not granted.\n")
	assert.True(t, IsForbidden(err))
}

func TestGraphQLCallOtherErrorsAreNotForbidden(t *testing.T) {
	cfg := answering(t, `{"errors": [{"message": "Validation error", "extensions": {"classification": "ValidationError"}}]}`)

	err := GraphQLCall(cfg, `{ projects { id } }`, nil, &struct{}{})

	assert.EqualError(t, err, "1) Validation error\n")
	assert.False(t, IsForbidden(err))
	assert.False(t, IsForbidden(errors.New("any")))
	assert.False(t, IsForbidden(nil))
}
