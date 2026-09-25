package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findingsCmdWithArgs parses args against the real 'validate findings' command.
func findingsCmdWithArgs(t *testing.T, args ...string) *cobra.Command {
	return cmdWithArgs(t, validateFindingsCmd, args...)
}

// writeReport writes a report file for the command to send.
func writeReport(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// marshalled renders a value the way the client sends it to Yontrack.
func marshalled(t *testing.T, value interface{}) string {
	t.Helper()
	body, err := json.Marshal(value)
	require.NoError(t, err)
	return string(body)
}

func TestValidateFindingsInput(t *testing.T) {
	t.Setenv("YONTRACK_BUILD_NAME", "")
	report := writeReport(t, `{"scanner": "zap", "kind": "DAST", "findings": []}`)
	cmd := findingsCmdWithArgs(t,
		"--project", "my-project", "--branch", "release/1.0", "--build", "42",
		"--validation", "DAST", "--description", "Nightly scan",
		"--format", "findings", "--kind", "DAST", "--scanner", "zap", "--report", report,
	)

	input, err := validateFindingsInput(cmd)

	require.NoError(t, err)
	assert.Equal(t, "my-project", input["project"])
	assert.Equal(t, "release-1.0", input["branch"])
	assert.Equal(t, "42", input["build"])
	assert.Equal(t, "DAST", input["validation"])
	assert.Equal(t, "Nightly scan", input["description"])
	assert.Equal(t, "findings", input["format"])
	assert.Equal(t, "DAST", input["kind"])
	assert.Equal(t, "zap", input["scanner"])
	// The report goes to Yontrack as JSON, not as a string holding JSON.
	assert.JSONEq(t, `{"scanner": "zap", "kind": "DAST", "findings": []}`, marshalled(t, input["report"]))
}

// Without --scanner, Yontrack takes the scanner from the report: an empty
// name must not reach it as an override.
func TestValidateFindingsInputWithoutScanner(t *testing.T) {
	report := writeReport(t, `{}`)
	cmd := findingsCmdWithArgs(t,
		"--project", "my-project", "--branch", "main", "--build", "42", "--validation", "IMAGE",
		"--format", "trivy", "--kind", "IMAGE", "--report", report,
	)

	input, err := validateFindingsInput(cmd)

	require.NoError(t, err)
	assert.NotContains(t, input, "scanner")
}

// The report travels as JSON inside the GraphQL request, so a file which is not
// JSON at all (an XML report, say) is refused before calling the API.
func TestValidateFindingsInputReportNotJSON(t *testing.T) {
	report := writeReport(t, `<xml/>`)
	cmd := findingsCmdWithArgs(t,
		"--project", "my-project", "--branch", "main", "--build", "42", "--validation", "CODE",
		"--format", "sarif", "--kind", "CODE", "--report", report,
	)

	_, err := validateFindingsInput(cmd)

	assert.EqualError(t, err, "the report "+report+" is not valid JSON")
}

// --format, --kind and --report are required, and a wrong kind is caught
// before calling the API, with the accepted values in the message.
func TestValidateFindingsInputFlagErrors(t *testing.T) {
	report := writeReport(t, `{}`)
	base := []string{"--project", "my-project", "--branch", "main", "--build", "42", "--validation", "SCAN"}
	cases := []struct {
		name  string
		args  []string
		error string
	}{
		{"missing format", []string{"--kind", "IMAGE", "--report", report},
			"format is required (use --format flag): findings, sarif, trivy"},
		{"missing kind", []string{"--format", "trivy", "--report", report},
			"kind is required (use --kind flag): IMAGE, CODE, SECRETS, DAST, DEPENDENCIES, OTHER"},
		{"unknown kind", []string{"--format", "trivy", "--kind", "image", "--report", report},
			"unknown kind image: IMAGE, CODE, SECRETS, DAST, DEPENDENCIES, OTHER"},
		{"missing report", []string{"--format", "trivy", "--kind", "IMAGE"},
			"report is required (use --report flag)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmd := findingsCmdWithArgs(t, append(base, c.args...)...)

			_, err := validateFindingsInput(cmd)

			assert.EqualError(t, err, c.error)
		})
	}
}

// The report reaches Yontrack as a JSON value inside the mutation input,
// exactly as it is in the file.
func TestValidateFindingsSendsReport(t *testing.T) {
	request := fakeYontrack(t, `{"data": {"validateBuildWithFindings": {"errors": []}}}`)
	report := writeReport(t, `{"scanner": "zap", "kind": "DAST", "findings": [{"externalId": "10038", "location": "", "severity": "MEDIUM", "title": "CSP"}]}`)
	cmd := findingsCmdWithArgs(t,
		"--project", "my-project", "--branch", "main", "--build", "42", "--validation", "DAST",
		"--format", "findings", "--kind", "DAST", "--report", report,
	)

	require.NoError(t, cmd.RunE(cmd, nil))

	assert.Contains(t, (*request)["query"], "validateBuildWithFindings(input: $input)")
	assert.JSONEq(t, `{
		"input": {
			"project": "my-project",
			"branch": "main",
			"build": "42",
			"validation": "DAST",
			"description": "",
			"runInfo": null,
			"format": "findings",
			"kind": "DAST",
			"report": {"scanner": "zap", "kind": "DAST", "findings": [{"externalId": "10038", "location": "", "severity": "MEDIUM", "title": "CSP"}]}
		}
	}`, marshalled(t, (*request)["variables"]))
}

// A report Yontrack rejects fails the command, with Yontrack's message.
func TestValidateFindingsFailsWithRejectedReport(t *testing.T) {
	fakeYontrack(t, `{"data": {"validateBuildWithFindings": {"errors": [{"message": "Unknown field: secret"}]}}}`)
	report := writeReport(t, `{"scanner": "gitleaks", "kind": "SECRETS", "findings": [{"secret": "hunter2"}]}`)
	cmd := findingsCmdWithArgs(t,
		"--project", "my-project", "--branch", "main", "--build", "42", "--validation", "SECRETS",
		"--format", "findings", "--kind", "SECRETS", "--report", report,
	)

	err := cmd.RunE(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown field: secret")
}

// Formats are left to Yontrack to check: one it adds must not wait for a
// release of the CLI.
func TestValidateFindingsInputFormatLeftToYontrack(t *testing.T) {
	report := writeReport(t, `{}`)
	cmd := findingsCmdWithArgs(t,
		"--project", "my-project", "--branch", "main", "--build", "42", "--validation", "SBOM",
		"--format", "cyclonedx", "--kind", "DEPENDENCIES", "--report", report,
	)

	input, err := validateFindingsInput(cmd)

	require.NoError(t, err)
	assert.Equal(t, "cyclonedx", input["format"])
}

// Against Yontrack 5.x, the mutation does not exist: the query is rejected as a
// whole, and the command fails with Yontrack's message.
func TestValidateFindingsFailsWithoutTheMutation(t *testing.T) {
	fakeYontrack(t, `{"errors": [{"message": "Validation error (UnknownType) : Unknown type 'ValidateBuildWithFindingsInput'"}]}`)
	report := writeReport(t, `{}`)
	cmd := findingsCmdWithArgs(t,
		"--project", "my-project", "--branch", "main", "--build", "42", "--validation", "IMAGE",
		"--format", "trivy", "--kind", "IMAGE", "--report", report,
	)

	err := cmd.RunE(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown type 'ValidateBuildWithFindingsInput'")
}
