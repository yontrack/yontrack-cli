package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

// cmdWithArgs parses args against a real command, and puts its flags back to
// their defaults once the test is over, since the command is shared by the
// whole package.
func cmdWithArgs(t *testing.T, cmd *cobra.Command, args ...string) *cobra.Command {
	t.Helper()
	t.Cleanup(func() {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	})
	require.NoError(t, cmd.ParseFlags(args))
	return cmd
}
