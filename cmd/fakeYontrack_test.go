package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	config "yontrack/config"

	"github.com/stretchr/testify/require"
)

// fakeYontrack answers every GraphQL request with the given response, and
// records the body of the last request. The CLI configuration is pointed at it
// for the duration of the test.
func fakeYontrack(t *testing.T, response string) *map[string]interface{} {
	t.Helper()
	request, _ := fakeYontrackWithEvidence(t, response, http.StatusCreated, `{}`)
	return request
}

// evidenceUpload is an evidence uploaded to the fake Yontrack.
type evidenceUpload struct {
	// Path of the request
	Path string
	// Name of the file, as sent in its part
	FileName string
	// Content of the file
	Content string
	// Other fields of the form
	Fields map[string]string
}

// fakeYontrackWithEvidence is fakeYontrack, which also accepts the uploads of
// evidence, answering them with the given status and body, and records them.
func fakeYontrackWithEvidence(t *testing.T, response string, evidenceStatus int, evidenceResponse string) (*map[string]interface{}, *[]evidenceUpload) {
	t.Helper()
	var request map[string]interface{}
	var uploads []evidenceUpload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/rest/extension/audit-trail/") {
			uploads = append(uploads, readEvidenceUpload(t, r))
			w.WriteHeader(evidenceStatus)
			_, _ = w.Write([]byte(evidenceResponse))
			return
		}
		request = nil
		_ = json.NewDecoder(r.Body).Decode(&request)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	pointConfigurationAt(t, server.URL)
	return &request, &uploads
}

// pointConfigurationAt points the CLI configuration at a fake Yontrack for the
// duration of the test.
func pointConfigurationAt(t *testing.T, url string) {
	t.Helper()
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(
		"selected: test\nconfigurations:\n  - name: test\n    url: "+url+"\n"), 0o600))
	previous := config.ConfigFilePath
	config.ConfigFilePath = configFile
	t.Cleanup(func() { config.ConfigFilePath = previous })
}

// graphQLRequest is a GraphQL request received by the fake Yontrack.
type graphQLRequest struct {
	Query     string
	Variables map[string]interface{}
	Header    http.Header
}

// fakeYontrackRoutes answers each GraphQL request with the response whose key
// its query contains - exactly one must - and records every request, headers
// included, in their order. It is for the commands which send several
// requests.
func fakeYontrackRoutes(t *testing.T, routes map[string]string) *[]graphQLRequest {
	t.Helper()
	var requests []graphQLRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query     string
			Variables map[string]interface{}
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		requests = append(requests, graphQLRequest{Query: body.Query, Variables: body.Variables, Header: r.Header.Clone()})
		var matches []string
		for key, response := range routes {
			if strings.Contains(body.Query, key) {
				matches = append(matches, response)
			}
		}
		if len(matches) != 1 {
			t.Errorf("%d routes match the query, expected 1:\n%s", len(matches), body.Query)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(matches[0]))
	}))
	t.Cleanup(server.Close)
	pointConfigurationAt(t, server.URL)
	return &requests
}

func readEvidenceUpload(t *testing.T, r *http.Request) evidenceUpload {
	upload := evidenceUpload{Path: r.URL.Path, Fields: map[string]string{}}
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		t.Errorf("the evidence is not sent as multipart/form-data: %v", err)
		return upload
	}
	for name, values := range r.MultipartForm.Value {
		upload.Fields[name] = values[0]
	}
	if files := r.MultipartForm.File["file"]; len(files) == 1 {
		upload.FileName = files[0].Filename
		file, err := files[0].Open()
		if err == nil {
			content, _ := io.ReadAll(file)
			upload.Content = string(content)
			_ = file.Close()
		}
	}
	return upload
}
