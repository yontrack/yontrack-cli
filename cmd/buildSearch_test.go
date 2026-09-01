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
	cmd.Flags().String("with-property", "", "")
	cmd.Flags().String("with-property-value", "", "")
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

func TestPropertyCriteria_None(t *testing.T) {
	propertyType, value, err := propertyCriteria("", "", "")

	assert.NoError(t, err)
	assert.Equal(t, "", propertyType)
	assert.Equal(t, "", value)
}

// --commit keeps working exactly as before: it is now one preset over the
// generic mechanism rather than the only thing it can express.
func TestPropertyCriteria_Commit(t *testing.T) {
	propertyType, value, err := propertyCriteria("abc123", "", "")

	assert.NoError(t, err)
	assert.Equal(t, gitCommitPropertyType, propertyType)
	assert.Equal(t, "abc123", value)
}

func TestPropertyCriteria_Generic(t *testing.T) {
	propertyType, value, err := propertyCriteria("", releasePropertyType, "5.3.0-rc-45")

	assert.NoError(t, err)
	assert.Equal(t, releasePropertyType, propertyType)
	assert.Equal(t, "5.3.0-rc-45", value)
}

// A property type on its own is a valid criterion: it matches any build
// carrying the property, whatever its value.
func TestPropertyCriteria_PropertyWithoutValue(t *testing.T) {
	propertyType, value, err := propertyCriteria("", releasePropertyType, "")

	assert.NoError(t, err)
	assert.Equal(t, releasePropertyType, propertyType)
	assert.Equal(t, "", value)
}

func TestPropertyCriteria_Conflicts(t *testing.T) {
	_, _, err := propertyCriteria("abc123", releasePropertyType, "")
	assert.EqualError(t, err, "--commit and --with-property are mutually exclusive")

	_, _, err = propertyCriteria("", "", "5.3.0-rc-45")
	assert.EqualError(t, err, "--with-property-value requires --with-property")
}

// The value must be left out of the form entirely when not given, rather than
// sent as an empty string, which the server would match against literally.
func TestFillFormWithProperty_PropertyWithoutValue(t *testing.T) {
	cmd := searchCmdWithFlags(map[string]string{"with-property": releasePropertyType})
	form := make(map[string]interface{})

	assert.NoError(t, fillFormWithProperty(cmd, &form, "property", "propertyValue"))
	assert.Equal(t, releasePropertyType, form["property"])
	assert.NotContains(t, form, "propertyValue")
}

func TestFillFormWithProperty_Commit(t *testing.T) {
	cmd := searchCmdWithFlags(map[string]string{"commit": "abc123"})
	form := make(map[string]interface{})

	assert.NoError(t, fillFormWithProperty(cmd, &form, "withProperty", "withPropertyValue"))
	assert.Equal(t, gitCommitPropertyType, form["withProperty"])
	assert.Equal(t, "abc123", form["withPropertyValue"])
}
