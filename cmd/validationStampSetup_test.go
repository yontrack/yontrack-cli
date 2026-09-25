package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 'vs setup' sets up the validation stamp itself, with an optional data type,
// as the README shows (issue #21).
func TestValidationStampSetupWithDataType(t *testing.T) {
	request := fakeYontrack(t, `{"data": {"setupValidationStamp": {"errors": []}}}`)
	cmd := cmdWithArgs(t, validationStampSetupCmd,
		"--project", "my-project", "--branch", "release/1.0", "--validation", "new",
		"--data-type", "net.nemerosa.ontrack.extension.general.validation.CHMLValidationDataType",
		"--data-config", `{warningLevel: {level: "HIGH",value:1},failedLevel:{level:"CRITICAL",value:1}}`,
	)

	require.NoError(t, cmd.RunE(cmd, nil))

	assert.Contains(t, (*request)["query"], `dataTypeConfig: {warningLevel: {level: "HIGH",value:1},failedLevel:{level:"CRITICAL",value:1}}`)
	variables := (*request)["variables"].(map[string]interface{})
	assert.Equal(t, "my-project", variables["project"])
	assert.Equal(t, "release-1.0", variables["branch"])
	assert.Equal(t, "new", variables["validation"])
	assert.Equal(t, "net.nemerosa.ontrack.extension.general.validation.CHMLValidationDataType", variables["dataType"])
}

// Without any data type, 'vs setup' creates a plain validation stamp.
func TestValidationStampSetupPlain(t *testing.T) {
	request := fakeYontrack(t, `{"data": {"setupValidationStamp": {"errors": []}}}`)
	cmd := cmdWithArgs(t, validationStampSetupCmd,
		"--project", "my-project", "--branch", "main", "--validation", "new",
	)

	require.NoError(t, cmd.RunE(cmd, nil))

	assert.Contains(t, (*request)["query"], "dataTypeConfig: null")
	variables := (*request)["variables"].(map[string]interface{})
	assert.Equal(t, "new", variables["validation"])
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
