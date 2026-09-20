package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Clears every environment variable used by the CI detection so that a test
// never depends on the CI engine the test suite itself runs in.
func clearCIEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"BITBUCKET_BUILD_NUMBER",
		"BITBUCKET_WORKSPACE",
		"BITBUCKET_REPO_SLUG",
		"BITBUCKET_COMMIT",
	} {
		t.Setenv(name, "")
	}
}

// Outside any known CI engine, nothing is defaulted.
func TestGetRunInfoDefaultsOutsideCI(t *testing.T) {
	clearCIEnv(t)

	defaults := GetRunInfoDefaults()

	assert.Equal(t, RunInfoDefaults{}, defaults)
}

// Inside Bitbucket Pipelines, the source and the trigger are guessed from the
// BITBUCKET_* variables.
func TestGetRunInfoDefaultsInBitbucketPipelines(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("BITBUCKET_BUILD_NUMBER", "42")
	t.Setenv("BITBUCKET_WORKSPACE", "my-workspace")
	t.Setenv("BITBUCKET_REPO_SLUG", "my-repository")
	t.Setenv("BITBUCKET_COMMIT", "abcdef1234567890")

	defaults := GetRunInfoDefaults()

	// Must match the RunInfoSourceTypeIcon registered in Yontrack
	// (yontrack/yontrack#1760).
	assert.Equal(t, "bitbucket-pipeline", defaults.SourceType)
	assert.Equal(t, "https://bitbucket.org/my-workspace/my-repository/pipelines/results/42", defaults.SourceURI)
	assert.Equal(t, "commit", defaults.TriggerType)
	assert.Equal(t, "abcdef1234567890", defaults.TriggerData)
}

// BITBUCKET_BUILD_NUMBER is the marker for Bitbucket Pipelines: without it, no
// default at all, even if the other variables happen to be set.
func TestGetRunInfoDefaultsWithoutBitbucketBuildNumber(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("BITBUCKET_WORKSPACE", "my-workspace")
	t.Setenv("BITBUCKET_REPO_SLUG", "my-repository")
	t.Setenv("BITBUCKET_COMMIT", "abcdef1234567890")

	defaults := GetRunInfoDefaults()

	assert.Equal(t, RunInfoDefaults{}, defaults)
}

// Without the workspace or the repository, no source URI can be built, but the
// source type and the trigger are still known.
func TestGetRunInfoDefaultsInBitbucketPipelinesWithoutRepository(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("BITBUCKET_BUILD_NUMBER", "42")
	t.Setenv("BITBUCKET_COMMIT", "abcdef1234567890")

	defaults := GetRunInfoDefaults()

	assert.Equal(t, "bitbucket-pipeline", defaults.SourceType)
	assert.Equal(t, "", defaults.SourceURI)
	assert.Equal(t, "commit", defaults.TriggerType)
	assert.Equal(t, "abcdef1234567890", defaults.TriggerData)
}
