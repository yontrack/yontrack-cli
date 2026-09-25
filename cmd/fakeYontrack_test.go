package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	config "yontrack/config"

	"github.com/stretchr/testify/require"
)

// fakeYontrack answers every GraphQL request with the given response, and
// records the body of the last request. The CLI configuration is pointed at it
// for the duration of the test.
func fakeYontrack(t *testing.T, response string) *map[string]interface{} {
	t.Helper()
	var request map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request = nil
		_ = json.NewDecoder(r.Body).Decode(&request)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)

	configFile := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(
		"selected: test\nconfigurations:\n  - name: test\n    url: "+server.URL+"\n"), 0o600))
	previous := config.ConfigFilePath
	config.ConfigFilePath = configFile
	t.Cleanup(func() { config.ConfigFilePath = previous })

	return &request
}
