package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// searchCmdWithFlags builds a command carrying the same search flags as
// `build search`, so the form helpers can be exercised the way the real command
// calls them.
func searchCmdWithFlags(flags map[string]string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("with-display-name", "", "")
	cmd.Flags().String("with-promotion", "", "")
	cmd.Flags().String("commit", "", "")
	cmd.Flags().Int("count", 10, "")
	for name, value := range flags {
		_ = cmd.Flags().Set(name, value)
	}
	return cmd
}

func TestFillFormWithDisplayName(t *testing.T) {
	cmd := searchCmdWithFlags(map[string]string{"with-display-name": "5.3.0-rc-45"})
	form := make(map[string]interface{})

	assert.NoError(t, fillFormWithDisplayName(cmd, &form, "withDisplayName"))
	assert.Equal(t, "5.3.0-rc-45", form["withDisplayName"])
}

// An absent flag must not put an empty criterion into the filter, which would
// match nothing instead of everything.
func TestFillFormWithDisplayName_Absent(t *testing.T) {
	cmd := searchCmdWithFlags(nil)
	form := make(map[string]interface{})

	assert.NoError(t, fillFormWithDisplayName(cmd, &form, "withDisplayName"))
	assert.NotContains(t, form, "withDisplayName")
}

// BuildSearchForm has no display-name field, so a project-wide search cannot
// honour the criterion. Failing is the only safe answer: dropping it silently
// would return builds that do not match what was asked for.
func TestRejectDisplayNameWithoutBranch(t *testing.T) {
	cmd := searchCmdWithFlags(map[string]string{"with-display-name": "5.3.0-rc-45"})

	assert.EqualError(t, rejectDisplayNameWithoutBranch(cmd),
		"--with-display-name requires --branch")
}

func TestRejectDisplayNameWithoutBranch_NotGiven(t *testing.T) {
	cmd := searchCmdWithFlags(nil)

	assert.NoError(t, rejectDisplayNameWithoutBranch(cmd))
}
