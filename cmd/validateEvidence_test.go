package cmd

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const createdValidationRun = `{"data": {"createValidationRun": {"validationRun": {"id": "55120"}, "errors": []}}}`

const evidenceRefused = `{"status": 503, "code": "audit-trail.evidence.storage-not-configured", "message": "No evidence storage is configured."}`

// writeEvidence writes a file to attach as evidence.
func writeEvidence(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// runValidate runs a 'validate' command with args, and returns its error and
// what it wrote on stderr.
func runValidate(t *testing.T, cmd *cobra.Command, args ...string) (error, string) {
	t.Helper()
	cmdWithArgs(t, cmd, args...)
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	t.Cleanup(func() { cmd.SetErr(nil) })
	err := cmd.RunE(cmd, nil)
	return err, stderr.String()
}

var validateArgs = []string{"--project", "my-project", "--branch", "main", "--build", "42", "--validation", "trivy", "--status", "PASSED"}

// The evidence is uploaded to the validation run which was just created, with
// what the tool which produced it.
func TestValidateUploadsEvidence(t *testing.T) {
	request, uploads := fakeYontrackWithEvidence(t, createdValidationRun, http.StatusCreated, `{"id": 7}`)
	report := writeEvidence(t, "trivy.pdf", "%PDF-1.7 report")
	sbom := writeEvidence(t, "sbom.json", `{"bomFormat": "CycloneDX"}`)

	err, stderr := runValidate(t, validateCmd, append(validateArgs,
		"--evidence", report, "--evidence", sbom,
		"--evidence-tool", "trivy", "--evidence-tool-version", "0.56.2",
		"--evidence-source-url", "https://ci.example.com/job/42")...)

	require.NoError(t, err)
	assert.Empty(t, stderr)
	assert.Contains(t, (*request)["query"], "validationRun {")
	require.Len(t, *uploads, 2)
	assert.Equal(t, evidenceUpload{
		Path:     "/rest/extension/audit-trail/validation-runs/55120/evidence",
		FileName: "trivy.pdf",
		Content:  "%PDF-1.7 report",
		Fields: map[string]string{
			"mediaType":      "application/pdf",
			"sourceTool":     "trivy",
			"sourceVersion":  "0.56.2",
			"sourceUrl":      "https://ci.example.com/job/42",
			"externalDigest": "sha256:336276927fc27d28c262f08eae5511abe9336bfb08fdeca8c7845c8a69d2e5e7",
		},
	}, (*uploads)[0])
	assert.Equal(t, "sbom.json", (*uploads)[1].FileName)
	assert.Equal(t, `{"bomFormat": "CycloneDX"}`, (*uploads)[1].Content)
	assert.Equal(t, "application/json", (*uploads)[1].Fields["mediaType"])
}

// Without --evidence, nothing is uploaded and the source flags are not needed.
func TestValidateWithoutEvidence(t *testing.T) {
	_, uploads := fakeYontrackWithEvidence(t, createdValidationRun, http.StatusCreated, `{}`)

	err, _ := runValidate(t, validateCmd, validateArgs...)

	require.NoError(t, err)
	assert.Empty(t, *uploads)
}

// A missing evidence is an audit gap: when Yontrack refuses it, the step
// fails - the validation run is recorded all the same.
func TestValidateFailsWhenTheEvidenceIsRefused(t *testing.T) {
	request, uploads := fakeYontrackWithEvidence(t, createdValidationRun, http.StatusServiceUnavailable, evidenceRefused)
	report := writeEvidence(t, "trivy.pdf", "%PDF-1.7 report")

	err, _ := runValidate(t, validateCmd, append(validateArgs, "--evidence", report)...)

	assert.EqualError(t, err, "evidence "+report+" was not attached to validation run 55120: "+
		"No evidence storage is configured. (audit-trail.evidence.storage-not-configured)")
	assert.NotNil(t, *request)
	assert.Len(t, *uploads, 1)
}

// With --evidence-optional, a refused evidence is only a warning.
func TestValidateWarnsWhenAnOptionalEvidenceIsRefused(t *testing.T) {
	_, uploads := fakeYontrackWithEvidence(t, createdValidationRun, http.StatusServiceUnavailable, evidenceRefused)
	report := writeEvidence(t, "trivy.pdf", "%PDF-1.7 report")

	err, stderr := runValidate(t, validateCmd, append(validateArgs, "--evidence", report, "--evidence-optional")...)

	require.NoError(t, err)
	assert.Equal(t, "Warning: evidence "+report+" was not attached to validation run 55120: "+
		"No evidence storage is configured. (audit-trail.evidence.storage-not-configured)\n", stderr)
	assert.Len(t, *uploads, 1)
}

// A refusal which is not one of an evidence - no right to upload, say - fails
// the step too, with what Yontrack answered.
func TestValidateFailsWhenTheUploadIsForbidden(t *testing.T) {
	fakeYontrackWithEvidence(t, createdValidationRun, http.StatusForbidden, `{"status": 403, "message": "Access denied"}`)
	report := writeEvidence(t, "trivy.pdf", "%PDF-1.7 report")

	err, _ := runValidate(t, validateCmd, append(validateArgs, "--evidence", report)...)

	assert.EqualError(t, err, "evidence "+report+" was not attached to validation run 55120: Access denied (HTTP 403)")
}

// Every evidence is tried, and every refusal reported.
func TestValidateReportsEveryRefusedEvidence(t *testing.T) {
	_, uploads := fakeYontrackWithEvidence(t, createdValidationRun, http.StatusServiceUnavailable, evidenceRefused)
	report := writeEvidence(t, "trivy.pdf", "%PDF-1.7 report")
	sbom := writeEvidence(t, "sbom.json", `{}`)

	err, _ := runValidate(t, validateCmd, append(validateArgs, "--evidence", report, "--evidence", sbom)...)

	require.Error(t, err)
	assert.Len(t, *uploads, 2)
	assert.Contains(t, err.Error(), "evidence "+report+" was not attached")
	assert.Contains(t, err.Error(), "evidence "+sbom+" was not attached")
}

// An evidence file which cannot be read fails the command before anything is
// recorded.
func TestValidateFailsOnAMissingEvidenceFile(t *testing.T) {
	request, uploads := fakeYontrackWithEvidence(t, createdValidationRun, http.StatusCreated, `{}`)
	missing := filepath.Join(t.TempDir(), "missing.pdf")

	err, _ := runValidate(t, validateCmd, append(validateArgs, "--evidence", missing)...)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read the evidence "+missing)
	assert.Nil(t, *request)
	assert.Empty(t, *uploads)
}

// With --evidence-optional, a missing evidence file is a warning, and the
// validation is recorded without it.
func TestValidateWarnsOnAMissingOptionalEvidenceFile(t *testing.T) {
	request, uploads := fakeYontrackWithEvidence(t, createdValidationRun, http.StatusCreated, `{}`)
	missing := filepath.Join(t.TempDir(), "missing.pdf")
	report := writeEvidence(t, "trivy.pdf", "%PDF-1.7 report")

	err, stderr := runValidate(t, validateCmd, append(validateArgs, "--evidence", missing, "--evidence", report, "--evidence-optional")...)

	require.NoError(t, err)
	assert.Contains(t, stderr, "Warning: cannot read the evidence "+missing)
	assert.NotNil(t, *request)
	require.Len(t, *uploads, 1)
	assert.Equal(t, "trivy.pdf", (*uploads)[0].FileName)
}

// The subcommands attach evidence too, to the validation run they create.
func TestValidateSubcommandUploadsEvidence(t *testing.T) {
	cases := []struct {
		name     string
		cmd      *cobra.Command
		response string
		args     []string
	}{
		{"tests", validateTestsCmd,
			`{"data": {"validateBuildWithTests": {"validationRun": {"id": "101"}, "errors": []}}}`,
			[]string{"--passed", "10"}},
		{"chml", validateCHMLCmd,
			`{"data": {"validateBuildWithCHML": {"validationRun": {"id": "101"}, "errors": []}}}`,
			[]string{"--high", "1"}},
		{"percentage", validatePercentageCmd,
			`{"data": {"validateBuildWithPercentage": {"validationRun": {"id": "101"}, "errors": []}}}`,
			[]string{"--value", "87"}},
		{"number", validateNumberCmd,
			`{"data": {"createValidationRun": {"validationRun": {"id": "101"}, "errors": []}}}`,
			[]string{"--value", "0"}},
		{"metrics", validateMetricsCmd,
			`{"data": {"validateBuildWithMetrics": {"validationRun": {"id": "101"}, "errors": []}}}`,
			[]string{"--metric", "speed=1.5"}},
		{"findings", validateFindingsCmd,
			`{"data": {"validateBuildWithFindings": {"validationRun": {"id": "101"}, "errors": []}}}`,
			[]string{"--format", "trivy", "--kind", "IMAGE", "--report", writeEvidence(t, "trivy.json", `{}`)}},
		{"junit", validateJUnitTestsCmd,
			`{"data": {"validateBuildWithTests": {"validationRun": {"id": "101"}, "errors": []}}}`,
			[]string{"--pattern", filepath.Join(t.TempDir(), "*.xml")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			request, uploads := fakeYontrackWithEvidence(t, c.response, http.StatusCreated, `{}`)
			evidence := writeEvidence(t, "report.txt", "report")

			err, _ := runValidate(t, c.cmd, append([]string{
				"--project", "my-project", "--branch", "main", "--build", "42", "--validation", "scan",
				"--evidence", evidence}, c.args...)...)

			require.NoError(t, err)
			assert.Contains(t, (*request)["query"], "validationRun {")
			require.Len(t, *uploads, 1)
			assert.Equal(t, "/rest/extension/audit-trail/validation-runs/101/evidence", (*uploads)[0].Path)
		})
	}
}
