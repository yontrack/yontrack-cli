package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chmlCmdWithArgs parses args against the real 'validate chml' command, and
// puts its flags back to their defaults once the test is over, since the
// command is shared by the whole package.
func chmlCmdWithArgs(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	t.Cleanup(func() {
		validateCHMLCmd.Flags().VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	})
	require.NoError(t, validateCHMLCmd.ParseFlags(args))
	return validateCHMLCmd
}

// In CI, the build comes from YONTRACK_BUILD_NAME, as for every other
// 'validate' subcommand (issue #73).
func TestValidateCHMLVariablesBuildFromEnv(t *testing.T) {
	t.Setenv("YONTRACK_BUILD_NAME", "42")
	cmd := chmlCmdWithArgs(t, "--project", "my-project", "--branch", "release/1.0", "--validation", "SECURITY", "--critical", "1")

	variables, err := validateCHMLVariables(cmd)

	require.NoError(t, err)
	assert.Equal(t, "my-project", variables["project"])
	assert.Equal(t, "release-1.0", variables["branch"])
	assert.Equal(t, "42", variables["build"])
	assert.Equal(t, "SECURITY", variables["validationStamp"])
	assert.Equal(t, 1, variables["critical"])
}

func TestValidateCHMLVariablesBuildFlagWinsOverEnv(t *testing.T) {
	t.Setenv("YONTRACK_BUILD_NAME", "42")
	cmd := chmlCmdWithArgs(t, "--project", "my-project", "--branch", "main", "--build", "43", "--validation", "SECURITY")

	variables, err := validateCHMLVariables(cmd)

	require.NoError(t, err)
	assert.Equal(t, "43", variables["build"])
}

// With no build at all, the command must fail before calling the API, instead
// of sending an empty build name.
func TestValidateCHMLVariablesBuildRequired(t *testing.T) {
	t.Setenv("YONTRACK_BUILD_NAME", "")
	cmd := chmlCmdWithArgs(t, "--project", "my-project", "--branch", "main", "--validation", "SECURITY")

	_, err := validateCHMLVariables(cmd)

	assert.EqualError(t, err, "build is required (use --build flag or YONTRACK_BUILD_NAME environment variable)")
}
