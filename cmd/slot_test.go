package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A slot that has never completed a deployment has no last deployed pipeline at
// all. Callers compare the deployed build against the one they are about to
// deploy, so this has to answer "" rather than panic.
func TestSlotDeployedBuild_NeverDeployed(t *testing.T) {
	s := slot{Id: "slot-1"}

	assert.Equal(t, "", s.DeployedBuildId())
	assert.Equal(t, "", s.DeployedBuildName())
}

// A pipeline can exist without a build in the payload we asked for; guard that
// too rather than assuming the shape.
func TestSlotDeployedBuild_PipelineWithoutBuild(t *testing.T) {
	s := slot{
		Id:                   "slot-1",
		LastDeployedPipeline: &slotPipelineRef{Id: "pipeline-1", Number: 7},
	}

	assert.Equal(t, "", s.DeployedBuildId())
	assert.Equal(t, "", s.DeployedBuildName())
}

func TestSlotDeployedBuild_Deployed(t *testing.T) {
	s := slot{
		Id: "slot-1",
		LastDeployedPipeline: &slotPipelineRef{
			Id:     "pipeline-1",
			Number: 7,
			Build: &slotBuild{
				Id:          "11407",
				Name:        "20260901055547-45",
				DisplayName: "5.3.0-rc-45",
			},
		},
	}

	assert.Equal(t, "11407", s.DeployedBuildId())
	assert.Equal(t, "5.3.0-rc-45", s.DeployedBuildName())
}

func TestCheckBuildSelector(t *testing.T) {
	assert.NoError(t, checkBuildSelector("1", ""))
	assert.NoError(t, checkBuildSelector("", "1.0.0"))

	assert.EqualError(t, checkBuildSelector("", ""),
		"one of --build or --version is required")
	assert.EqualError(t, checkBuildSelector("1", "1.0.0"),
		"--build and --version are mutually exclusive")
}

// buildId is an Int in StartSlotPipelineInput. Passing it as a string is
// rejected by the server, so it has to stay a number all the way through.
func TestStartSlotPipelineInput(t *testing.T) {
	input := startSlotPipelineInput("slot-1", 11407)

	assert.Equal(t, "slot-1", input["slotId"])
	assert.Equal(t, 11407, input["buildId"])
	assert.IsType(t, 0, input["buildId"])
}
