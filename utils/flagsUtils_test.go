package utils

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestGetBuildIdFromEnv(t *testing.T) {
	// Not set
	_ = os.Unsetenv("YONTRACK_BUILD_ID")
	val, _ := GetBuildIdFromEnv()
	assert.Equal(t, 0, val)

	// Set to empty
	_ = os.Setenv("YONTRACK_BUILD_ID", "")
	val, _ = GetBuildIdFromEnv()
	assert.Equal(t, 0, val)

	// Set to non-integer
	_ = os.Setenv("YONTRACK_BUILD_ID", "abc")
	val, _ = GetBuildIdFromEnv()
	assert.Equal(t, 0, val)

	// Set to integer
	_ = os.Setenv("YONTRACK_BUILD_ID", "123")
	val, _ = GetBuildIdFromEnv()
	assert.Equal(t, 123, val)
}

// dateCmd is a command with a --date flag, parsed from args.
func dateCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("date", "", "")
	assert.NoError(t, cmd.ParseFlags(args))
	return cmd
}

// Without --date, nothing is backdated: Yontrack takes the current time.
func TestGetDateFlag_NotSet(t *testing.T) {
	date, err := GetDateFlag(dateCmd(t))

	assert.NoError(t, err)
	assert.Equal(t, "", date)
}

// Yontrack reads a date/time without an offset as UTC, so the date is sent
// that way, whatever offset it was given with.
func TestGetDateFlag(t *testing.T) {
	cases := map[string]string{
		"2026-09-15T14:30:00Z":      "2026-09-15T14:30:00",
		"2026-09-15T14:30:00+02:00": "2026-09-15T12:30:00",
		"2026-09-15T14:30:00.250Z":  "2026-09-15T14:30:00",
		"2026-09-15T14:30:00":       "2026-09-15T14:30:00",
		"2026-09-15":                "2026-09-15T00:00:00",
	}
	for given, sent := range cases {
		t.Run(given, func(t *testing.T) {
			date, err := GetDateFlag(dateCmd(t, "--date", given))

			assert.NoError(t, err)
			assert.Equal(t, sent, date)
		})
	}
}

func TestGetDateFlag_Invalid(t *testing.T) {
	_, err := GetDateFlag(dateCmd(t, "--date", "15/09/2026"))

	assert.EqualError(t, err, "invalid --date 15/09/2026: expected 2006-01-02, 2006-01-02T15:04:05 (UTC) or 2006-01-02T15:04:05Z07:00")
}

// pipelineCmd is a command with a --pipeline flag, parsed from args.
func pipelineCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("pipeline", "", "")
	assert.NoError(t, cmd.ParseFlags(args))
	return cmd
}

func TestGetPipelineFlag(t *testing.T) {
	t.Setenv("YONTRACK_PIPELINE_ID", "from-env")

	pipeline, err := GetPipelineFlag(pipelineCmd(t, "--pipeline", "from-flag"))

	assert.NoError(t, err)
	assert.Equal(t, "from-flag", pipeline)
}

// 'slot pipeline start --output env' exports YONTRACK_PIPELINE_ID, so a later
// step of the same job does not have to pass it again.
func TestGetPipelineFlag_FromEnv(t *testing.T) {
	t.Setenv("YONTRACK_PIPELINE_ID", "from-env")

	pipeline, err := GetPipelineFlag(pipelineCmd(t))

	assert.NoError(t, err)
	assert.Equal(t, "from-env", pipeline)
}

func TestGetPipelineFlag_Missing(t *testing.T) {
	t.Setenv("YONTRACK_PIPELINE_ID", "")

	_, err := GetPipelineFlag(pipelineCmd(t))

	assert.EqualError(t, err, "pipeline is required (use --pipeline flag or YONTRACK_PIPELINE_ID environment variable)")
}
