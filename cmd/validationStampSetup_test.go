package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 'vs setup' sets up the validation stamp itself, with an optional data type,
// as the README shows (issue #21). The configuration is plain JSON, and travels
// as a variable: nothing is spliced into the query (issue #80).
func TestValidationStampSetupWithDataType(t *testing.T) {
	request := fakeYontrack(t, `{"data": {"setupValidationStamp": {"errors": []}}}`)
	cmd := cmdWithArgs(t, validationStampSetupCmd,
		"--project", "my-project", "--branch", "release/1.0", "--validation", "new",
		"--data-type", "net.nemerosa.ontrack.extension.findings.validation.FindingsValidationDataType",
		"--data-config", `{"warningLevel":"HIGH","warningValue":1,"failedLevel":"CRITICAL","failedValue":1}`,
	)

	require.NoError(t, cmd.RunE(cmd, nil))

	query := (*request)["query"].(string)
	assert.Contains(t, query, "$dataTypeConfig: JSON")
	assert.Contains(t, query, "dataTypeConfig: $dataTypeConfig")
	assert.NotContains(t, query, "warningLevel")
	variables := (*request)["variables"].(map[string]interface{})
	assert.Equal(t, "my-project", variables["project"])
	assert.Equal(t, "release-1.0", variables["branch"])
	assert.Equal(t, "new", variables["validation"])
	assert.Equal(t, "net.nemerosa.ontrack.extension.findings.validation.FindingsValidationDataType", variables["dataType"])
	// An object, not a string holding one
	assert.Equal(t, map[string]interface{}{
		"warningLevel": "HIGH",
		"warningValue": float64(1),
		"failedLevel":  "CRITICAL",
		"failedValue":  float64(1),
	}, variables["dataTypeConfig"])
}

// 'vs setup generic' goes the same way as 'vs setup'.
func TestValidationStampSetupGenericSendsConfigAsVariable(t *testing.T) {
	request := fakeYontrack(t, `{"data": {"setupValidationStamp": {"errors": []}}}`)
	cmd := cmdWithArgs(t, validationStampSetupGenericCmd,
		"--project", "my-project", "--branch", "main", "--validation", "new",
		"--data-type", "net.nemerosa.ontrack.extension.general.validation.CHMLValidationDataType",
		"--data-config", `{"warningLevel":"HIGH","warningValue":1,"failedLevel":"CRITICAL","failedValue":1}`,
	)

	require.NoError(t, cmd.RunE(cmd, nil))

	assert.NotContains(t, (*request)["query"], "warningLevel")
	variables := (*request)["variables"].(map[string]interface{})
	assert.Equal(t, "HIGH", variables["dataTypeConfig"].(map[string]interface{})["warningLevel"])
}

// A malformed --data-config fails before any call is made.
func TestValidationStampSetupRejectsInvalidDataConfig(t *testing.T) {
	request := fakeYontrack(t, `{"data": {"setupValidationStamp": {"errors": []}}}`)
	cmd := cmdWithArgs(t, validationStampSetupCmd,
		"--project", "my-project", "--branch", "main", "--validation", "new",
		"--data-type", "net.nemerosa.ontrack.extension.general.validation.CHMLValidationDataType",
		"--data-config", `{warningLevel: "HIGH"}`,
	)

	err := cmd.RunE(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--data-config is not valid JSON")
	assert.Nil(t, *request, "no call must have been made")
}

// Without any data type, 'vs setup' creates a plain validation stamp, and the
// configuration is sent as null.
func TestValidationStampSetupPlain(t *testing.T) {
	request := fakeYontrack(t, `{"data": {"setupValidationStamp": {"errors": []}}}`)
	cmd := cmdWithArgs(t, validationStampSetupCmd,
		"--project", "my-project", "--branch", "main", "--validation", "new",
	)

	require.NoError(t, cmd.RunE(cmd, nil))

	assert.Contains(t, (*request)["query"], "dataTypeConfig: $dataTypeConfig")
	variables := (*request)["variables"].(map[string]interface{})
	assert.Equal(t, "new", variables["validation"])
	assert.Contains(t, variables, "dataTypeConfig")
	assert.Nil(t, variables["dataTypeConfig"])
}

// A validation stamp Yontrack rejects fails the command, with Yontrack's message.
func TestValidationStampSetupFailsWithRejectedDataType(t *testing.T) {
	fakeYontrack(t, `{"data": {"setupValidationStamp": {"errors": [{"message": "Unknown data type: Foo"}]}}}`)
	cmd := cmdWithArgs(t, validationStampSetupCmd,
		"--project", "my-project", "--branch", "main", "--validation", "new", "--data-type", "Foo",
	)

	err := cmd.RunE(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown data type: Foo")
}

// The dedicated commands per data type are still reached through 'vs setup'.
func TestValidationStampSetupKeepsItsSubcommands(t *testing.T) {
	for name, expected := range map[string]*cobra.Command{
		"chml":       validationStampSetupCHMLCmd,
		"generic":    validationStampSetupGenericCmd,
		"metrics":    validationStampSetupMetricsCmd,
		"percentage": validationStampSetupPercentageCmd,
		"tests":      validationStampSetupTestsCmd,
	} {
		found, _, err := validationStampSetupCmd.Find([]string{name})
		require.NoError(t, err)
		assert.Same(t, expected, found, name)
	}
}

// A mistyped subcommand is an error, not a plain validation stamp setup.
func TestValidationStampSetupRejectsUnknownSubcommand(t *testing.T) {
	err := validationStampSetupCmd.ValidateArgs([]string{"chmll"})

	assert.Error(t, err)
}

// The data type flags belong to 'vs setup' only: the dedicated commands do not
// take them.
func TestValidationStampSetupDataTypeFlagsNotInherited(t *testing.T) {
	assert.Nil(t, validationStampSetupCHMLCmd.InheritedFlags().Lookup("data-type"))
	assert.Nil(t, validationStampSetupCHMLCmd.InheritedFlags().Lookup("data-config"))
}
