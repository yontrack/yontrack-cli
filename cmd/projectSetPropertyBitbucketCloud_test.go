package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The Bitbucket Cloud workspace moved from the configuration to the project
// property (yontrack/yontrack#1756), so it must be sent with every call.
func TestBitbucketCloudPropertyVariables(t *testing.T) {
	variables := bitbucketCloudPropertyVariables("my-project", "Bitbucket-Cloud", "my-workspace", "my-repository", 30, "jira//my-jira")

	assert.Equal(t, "my-project", variables["project"])
	assert.Equal(t, "Bitbucket-Cloud", variables["configuration"])
	assert.Equal(t, "my-workspace", variables["workspace"])
	assert.Equal(t, "my-repository", variables["repository"])
	assert.Equal(t, 30, variables["indexationInterval"])
	assert.Equal(t, "jira//my-jira", variables["issueServiceConfigurationIdentifier"])
}

// The repository is the plain repository slug: the workspace is carried by its
// own field and must not be prefixed onto the repository.
func TestBitbucketCloudPropertyVariablesKeepsWorkspaceAndRepositorySeparate(t *testing.T) {
	variables := bitbucketCloudPropertyVariables("my-project", "Bitbucket-Cloud", "my-workspace", "my-repository", 0, "")

	assert.Equal(t, "my-workspace", variables["workspace"])
	assert.Equal(t, "my-repository", variables["repository"])
}
