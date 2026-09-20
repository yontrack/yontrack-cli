package cmd

import (
	"testing"
	"yontrack/client"
	"yontrack/utils"

	"github.com/stretchr/testify/assert"
)

var bitbucketDefaults = utils.RunInfoDefaults{
	SourceType:  "bitbucket-pipeline",
	SourceURI:   "https://bitbucket.org/my-workspace/my-repository/pipelines/results/42",
	TriggerType: "commit",
	TriggerData: "abcdef1234567890",
}

var gitlabDefaults = utils.RunInfoDefaults{
	SourceType:  "gitlab-pipeline",
	SourceURI:   "https://gitlab.com/my-group/my-project/-/pipelines/42",
	TriggerType: "commit",
	TriggerData: "abcdef1234567890",
}

// Inside a CI engine, with no flag given, every field is defaulted from the
// environment.
func TestApplyRunInfoDefaultsWithoutFlags(t *testing.T) {
	info := applyRunInfoDefaults(client.RunInfo{}, bitbucketDefaults)

	assert.Equal(t, "bitbucket-pipeline", info.SourceType)
	assert.Equal(t, "https://bitbucket.org/my-workspace/my-repository/pipelines/results/42", info.SourceURI)
	assert.Equal(t, "commit", info.TriggerType)
	assert.Equal(t, "abcdef1234567890", info.TriggerData)
}

// Explicit flags always win over the CI defaults.
func TestApplyRunInfoDefaultsWithExplicitFlags(t *testing.T) {
	info := applyRunInfoDefaults(client.RunInfo{
		SourceType:  "jenkins",
		SourceURI:   "https://jenkins/job/my-job/1",
		TriggerType: "scm",
		TriggerData: "0123456789abcdef",
		RunTime:     80,
	}, bitbucketDefaults)

	assert.Equal(t, "jenkins", info.SourceType)
	assert.Equal(t, "https://jenkins/job/my-job/1", info.SourceURI)
	assert.Equal(t, "scm", info.TriggerType)
	assert.Equal(t, "0123456789abcdef", info.TriggerData)
	assert.Equal(t, 80, info.RunTime)
}

// Defaulting is per field: only the fields which were not given are filled in.
func TestApplyRunInfoDefaultsMixesFlagsAndDefaults(t *testing.T) {
	info := applyRunInfoDefaults(client.RunInfo{
		SourceURI: "https://jenkins/job/my-job/1",
	}, bitbucketDefaults)

	assert.Equal(t, "bitbucket-pipeline", info.SourceType)
	assert.Equal(t, "https://jenkins/job/my-job/1", info.SourceURI)
	assert.Equal(t, "commit", info.TriggerType)
	assert.Equal(t, "abcdef1234567890", info.TriggerData)
}

// The defaulting is engine-agnostic: inside GitLab CI, with no flag given,
// every field is defaulted from the environment.
func TestApplyRunInfoDefaultsWithoutFlagsInGitLabCI(t *testing.T) {
	info := applyRunInfoDefaults(client.RunInfo{}, gitlabDefaults)

	assert.Equal(t, "gitlab-pipeline", info.SourceType)
	assert.Equal(t, "https://gitlab.com/my-group/my-project/-/pipelines/42", info.SourceURI)
	assert.Equal(t, "commit", info.TriggerType)
	assert.Equal(t, "abcdef1234567890", info.TriggerData)
}

// Explicit flags win over the GitLab CI defaults too.
func TestApplyRunInfoDefaultsWithExplicitFlagsInGitLabCI(t *testing.T) {
	info := applyRunInfoDefaults(client.RunInfo{
		SourceType:  "jenkins",
		SourceURI:   "https://jenkins/job/my-job/1",
		TriggerType: "scm",
		TriggerData: "0123456789abcdef",
		RunTime:     80,
	}, gitlabDefaults)

	assert.Equal(t, "jenkins", info.SourceType)
	assert.Equal(t, "https://jenkins/job/my-job/1", info.SourceURI)
	assert.Equal(t, "scm", info.TriggerType)
	assert.Equal(t, "0123456789abcdef", info.TriggerData)
	assert.Equal(t, 80, info.RunTime)
}

// Outside any known CI engine, nothing changes.
func TestApplyRunInfoDefaultsOutsideCI(t *testing.T) {
	info := applyRunInfoDefaults(client.RunInfo{RunTime: 80}, utils.RunInfoDefaults{})

	assert.Equal(t, client.RunInfo{RunTime: 80}, info)
}

// The run info is only omitted when neither a flag nor a CI default provides
// anything.
func TestIsRunInfoEmpty(t *testing.T) {
	assert.True(t, isRunInfoEmpty(client.RunInfo{}))
	assert.False(t, isRunInfoEmpty(client.RunInfo{SourceType: "bitbucket-pipeline"}))
	assert.False(t, isRunInfoEmpty(client.RunInfo{SourceURI: "https://bitbucket.org/w/r/pipelines/results/42"}))
	assert.False(t, isRunInfoEmpty(client.RunInfo{TriggerType: "commit"}))
	assert.False(t, isRunInfoEmpty(client.RunInfo{TriggerData: "abcdef"}))
	assert.False(t, isRunInfoEmpty(client.RunInfo{RunTime: 80}))
}
