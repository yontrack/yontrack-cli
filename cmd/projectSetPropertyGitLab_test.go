package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The GitLab project property carries the same field set as the GitHub one:
// unlike Bitbucket Cloud, there is no workspace to split out.
func TestGitLabPropertyVariables(t *testing.T) {
	variables := gitlabPropertyVariables("my-project", "gitlab.com", "my-group/my-project", 30, "jira//my-jira")

	assert.Equal(t, "my-project", variables["project"])
	assert.Equal(t, "gitlab.com", variables["configuration"])
	assert.Equal(t, "my-group/my-project", variables["repository"])
	assert.Equal(t, 30, variables["indexationInterval"])
	assert.Equal(t, "jira//my-jira", variables["issueServiceConfigurationIdentifier"])
}

// A GitLab project path can be arbitrarily deep: subgroups are normal, so the
// repository must be passed through untouched, slashes included.
func TestGitLabPropertyVariablesKeepsSubgroupPath(t *testing.T) {
	variables := gitlabPropertyVariables("my-project", "gitlab.com", "my-group/my-subgroup/my-project", 0, "")

	assert.Equal(t, "my-group/my-subgroup/my-project", variables["repository"])
	assert.Equal(t, 0, variables["indexationInterval"])
	assert.Equal(t, "", variables["issueServiceConfigurationIdentifier"])
}
