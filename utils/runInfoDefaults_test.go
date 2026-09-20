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
		"GITLAB_CI",
		"CI_PIPELINE_URL",
		"CI_COMMIT_SHA",
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

// Inside GitLab CI, the source and the trigger are guessed from the CI_*
// variables.
func TestGetRunInfoDefaultsInGitLabCI(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("GITLAB_CI", "true")
	t.Setenv("CI_PIPELINE_URL", "https://gitlab.com/my-group/my-project/-/pipelines/42")
	t.Setenv("CI_COMMIT_SHA", "abcdef1234567890")

	defaults := GetRunInfoDefaults()

	// Must match the RunInfoSourceTypeIcon registered in Yontrack for the
	// gitlab-ci CI engine.
	assert.Equal(t, "gitlab-pipeline", defaults.SourceType)
	assert.Equal(t, "https://gitlab.com/my-group/my-project/-/pipelines/42", defaults.SourceURI)
	assert.Equal(t, "commit", defaults.TriggerType)
	assert.Equal(t, "abcdef1234567890", defaults.TriggerData)
}

// GITLAB_CI is the marker for GitLab CI: without it, no default at all, even if
// the other variables happen to be set.
func TestGetRunInfoDefaultsWithoutGitLabCI(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("CI_PIPELINE_URL", "https://gitlab.com/my-group/my-project/-/pipelines/42")
	t.Setenv("CI_COMMIT_SHA", "abcdef1234567890")

	defaults := GetRunInfoDefaults()

	assert.Equal(t, RunInfoDefaults{}, defaults)
}

// GitLab always sets GITLAB_CI to "true": any other value is not GitLab CI.
func TestGetRunInfoDefaultsWithGitLabCINotTrue(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("GITLAB_CI", "false")
	t.Setenv("CI_PIPELINE_URL", "https://gitlab.com/my-group/my-project/-/pipelines/42")
	t.Setenv("CI_COMMIT_SHA", "abcdef1234567890")

	defaults := GetRunInfoDefaults()

	assert.Equal(t, RunInfoDefaults{}, defaults)
}

// Without CI_PIPELINE_URL, there is no source URI, but the source type and the
// trigger are still known.
func TestGetRunInfoDefaultsInGitLabCIWithoutPipelineURL(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("GITLAB_CI", "true")
	t.Setenv("CI_COMMIT_SHA", "abcdef1234567890")

	defaults := GetRunInfoDefaults()

	assert.Equal(t, "gitlab-pipeline", defaults.SourceType)
	assert.Equal(t, "", defaults.SourceURI)
	assert.Equal(t, "commit", defaults.TriggerType)
	assert.Equal(t, "abcdef1234567890", defaults.TriggerData)
}

// Without CI_COMMIT_SHA, there is no trigger data; the trigger type is still
// "commit", as it is for Bitbucket Pipelines.
func TestGetRunInfoDefaultsInGitLabCIWithoutCommit(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("GITLAB_CI", "true")
	t.Setenv("CI_PIPELINE_URL", "https://gitlab.com/my-group/my-project/-/pipelines/42")

	defaults := GetRunInfoDefaults()

	assert.Equal(t, "gitlab-pipeline", defaults.SourceType)
	assert.Equal(t, "https://gitlab.com/my-group/my-project/-/pipelines/42", defaults.SourceURI)
	assert.Equal(t, "commit", defaults.TriggerType)
	assert.Equal(t, "", defaults.TriggerData)
}

// Two CI engines can never be detected at the same time for real, but the
// outcome must not depend on the map order of the environment: the engines are
// tried in a fixed order and Bitbucket Pipelines, detected first, wins. Keeping
// Bitbucket first means adding GitLab cannot change what an existing Bitbucket
// Pipelines user gets.
func TestGetRunInfoDefaultsWithBothEnginesDetected(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("BITBUCKET_BUILD_NUMBER", "42")
	t.Setenv("BITBUCKET_WORKSPACE", "my-workspace")
	t.Setenv("BITBUCKET_REPO_SLUG", "my-repository")
	t.Setenv("BITBUCKET_COMMIT", "abcdef1234567890")
	t.Setenv("GITLAB_CI", "true")
	t.Setenv("CI_PIPELINE_URL", "https://gitlab.com/my-group/my-project/-/pipelines/42")
	t.Setenv("CI_COMMIT_SHA", "0123456789abcdef")

	defaults := GetRunInfoDefaults()

	assert.Equal(t, "bitbucket-pipeline", defaults.SourceType)
	assert.Equal(t, "https://bitbucket.org/my-workspace/my-repository/pipelines/results/42", defaults.SourceURI)
}
