package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
	config "yontrack/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	input := startSlotPipelineInput("slot-1", 11407, "")

	assert.Equal(t, "slot-1", input["slotId"])
	assert.Equal(t, 11407, input["buildId"])
	assert.IsType(t, 0, input["buildId"])
	// Not backdated: Yontrack takes the current time.
	assert.NotContains(t, input, "dateTime")
}

// --date backdates the start of the pipeline.
func TestStartSlotPipelineInput_Backdated(t *testing.T) {
	input := startSlotPipelineInput("slot-1", 11407, "2026-09-15T14:30:00")

	assert.Equal(t, "2026-09-15T14:30:00", input["dateTime"])
}

func boolPtr(b bool) *bool { return &b }

// A pipeline whose admission rules are not all met is still created, as a
// candidate. Nothing on the server says so, so the CLI has to (issue #77).
func TestNotDeployableWarning(t *testing.T) {
	pipeline := startedSlotPipeline{
		slotPipelineRef: slotPipelineRef{Id: "p1", Number: 3},
		RunAction:       &slotPipelineRunAction{Ok: boolPtr(false)},
		AdmissionRules: []slotPipelineAdmissionRule{
			{
				AdmissionRuleConfig: slotAdmissionRuleConfig{Name: "promotion", RuleId: "promotion"},
				Check:               slotDeploymentCheck{Ok: boolPtr(false), Reason: "Build not promoted"},
			},
			{
				AdmissionRuleConfig: slotAdmissionRuleConfig{Name: "branch", RuleId: "branch"},
				Check:               slotDeploymentCheck{Ok: boolPtr(true)},
			},
			{
				AdmissionRuleConfig: slotAdmissionRuleConfig{Name: "approval", RuleId: "manual"},
				Check:               slotDeploymentCheck{Ok: boolPtr(false), Reason: "Not approved"},
				Overridden:          true,
			},
		},
	}

	assert.Equal(t,
		"Pipeline #3 created as a candidate — not deployable yet:\n"+
			"  - promotion: Build not promoted\n",
		notDeployableWarning(&pipeline))
}

// Without a named rule, the rule ID says which one it is.
func TestNotDeployableWarning_RuleWithoutName(t *testing.T) {
	pipeline := startedSlotPipeline{
		slotPipelineRef: slotPipelineRef{Id: "p1", Number: 3},
		RunAction:       &slotPipelineRunAction{Ok: boolPtr(false)},
		AdmissionRules: []slotPipelineAdmissionRule{
			{
				AdmissionRuleConfig: slotAdmissionRuleConfig{RuleId: "promotion"},
				Check:               slotDeploymentCheck{Ok: boolPtr(false), Reason: "Build not promoted"},
			},
		},
	}

	assert.Contains(t, notDeployableWarning(&pipeline), "  - promotion: Build not promoted\n")
}

func TestNotDeployableWarning_Deployable(t *testing.T) {
	pipeline := startedSlotPipeline{
		slotPipelineRef: slotPipelineRef{Id: "p1", Number: 3},
		RunAction:       &slotPipelineRunAction{Ok: boolPtr(true)},
	}

	assert.Equal(t, "", notDeployableWarning(&pipeline))
}

// No run action at all tells nothing either way: no warning.
func TestNotDeployableWarning_Unknown(t *testing.T) {
	pipeline := startedSlotPipeline{slotPipelineRef: slotPipelineRef{Id: "p1", Number: 3}}

	assert.Equal(t, "", notDeployableWarning(&pipeline))
}

// The started pipeline carries what tells whether it can run, and its JSON
// output stays what it was.
func TestStartSlotPipelineReadsRunAction(t *testing.T) {
	request := fakeYontrack(t, `{"data": {"startSlotPipeline": {"pipeline": {
		"id": "p1", "number": 3,
		"runAction": {"ok": false},
		"admissionRules": [{"admissionRuleConfig": {"name": "promotion", "ruleId": "promotion"}, "check": {"ok": false, "reason": "Build not promoted"}, "overridden": false}]
	}, "errors": []}}}`)
	cfg, err := config.GetSelectedConfiguration()
	require.NoError(t, err)

	pipeline, err := startSlotPipeline(cfg, "slot-1", 11407, "")

	require.NoError(t, err)
	query := (*request)["query"].(string)
	assert.Contains(t, query, "runAction")
	assert.Contains(t, query, "admissionRules")
	assert.Contains(t, notDeployableWarning(pipeline), "promotion: Build not promoted")
	body, err := json.Marshal(pipeline)
	require.NoError(t, err)
	assert.JSONEq(t, `{"id": "p1", "number": 3, "build": null}`, string(body))
}

// Eligible builds are the deployable ones by default. 'deployable' is always
// passed, never left to the server default, which changed (issue #77).
func TestSlotBuildsDeployableByDefault(t *testing.T) {
	request := fakeYontrack(t, `{"data": {"environmentByName": {"slots": [{"id": "slot-1", "eligibleBuilds": {"pageItems": [
		{"id": "11407", "name": "45", "displayName": "5.3.0-rc-45"},
		{"id": "11406", "name": "44", "displayName": "5.3.0-rc-44"}
	]}}]}}}`)
	cmd := cmdWithArgs(t, slotBuildsCmd, "--project", "my-project", "--environment", "production")
	var out bytes.Buffer
	cmd.SetOut(&out)
	t.Cleanup(func() { cmd.SetOut(nil) })

	require.NoError(t, cmd.RunE(cmd, nil))

	variables := (*request)["variables"].(map[string]interface{})
	assert.Equal(t, "my-project", variables["project"])
	assert.Equal(t, "production", variables["environment"])
	assert.Equal(t, true, variables["deployable"])
	assert.Equal(t, float64(10), variables["size"])
	assert.Equal(t, "45\n44\n", out.String())
}

// --all lists every eligible build, including those not deployable yet.
func TestSlotBuildsAll(t *testing.T) {
	request := fakeYontrack(t, `{"data": {"environmentByName": {"slots": [{"id": "slot-1", "eligibleBuilds": {"pageItems": []}}]}}}`)
	cmd := cmdWithArgs(t, slotBuildsCmd, "--project", "my-project", "--environment", "production", "--all", "--count", "5")
	cmd.SetOut(&bytes.Buffer{})
	t.Cleanup(func() { cmd.SetOut(nil) })

	require.NoError(t, cmd.RunE(cmd, nil))

	variables := (*request)["variables"].(map[string]interface{})
	assert.Equal(t, false, variables["deployable"])
	assert.Equal(t, float64(5), variables["size"])
}

func TestSlotBuildsJSON(t *testing.T) {
	fakeYontrack(t, `{"data": {"environmentByName": {"slots": [{"id": "slot-1", "eligibleBuilds": {"pageItems": [
		{"id": "11407", "name": "45", "displayName": "5.3.0-rc-45"}
	]}}]}}}`)
	cmd := cmdWithArgs(t, slotBuildsCmd, "--project", "my-project", "--environment", "production", "--output", "json")
	var out bytes.Buffer
	cmd.SetOut(&out)
	t.Cleanup(func() { cmd.SetOut(nil) })

	require.NoError(t, cmd.RunE(cmd, nil))

	assert.JSONEq(t, `[{"id": "11407", "name": "45", "displayName": "5.3.0-rc-45"}]`, out.String())
}

func TestSlotBuildsNoSlot(t *testing.T) {
	fakeYontrack(t, `{"data": {"environmentByName": {"slots": []}}}`)
	cmd := cmdWithArgs(t, slotBuildsCmd, "--project", "my-project", "--environment", "production")

	assert.EqualError(t, cmd.RunE(cmd, nil), "no slot for project my-project in environment production")
}

// slotPipelineFailInput is what 'slot pipeline fail' sent to failSlotPipeline.
func slotPipelineFailInput(t *testing.T, request *map[string]interface{}) map[string]interface{} {
	t.Helper()
	assert.Contains(t, (*request)["query"].(string), "failSlotPipeline")
	variables := (*request)["variables"].(map[string]interface{})
	return variables["input"].(map[string]interface{})
}

func TestSlotPipelineFail(t *testing.T) {
	request := fakeYontrack(t, `{"data": {"failSlotPipeline": {"failStatus": {"ok": true, "message": "Smoke tests failed"}, "errors": []}}}`)
	cmd := cmdWithArgs(t, slotPipelineFailCmd,
		"--pipeline", "p1", "--message", "Smoke tests failed", "--date", "2026-09-15T14:30:00+02:00")

	require.NoError(t, cmd.RunE(cmd, nil))

	assert.Equal(t, map[string]interface{}{
		"pipelineId": "p1",
		"message":    "Smoke tests failed",
		"dateTime":   "2026-09-15T12:30:00",
	}, slotPipelineFailInput(t, request))
}

// Yontrack refuses to fail a pipeline which is not running through the status
// of the action, not through the errors: the command must fail all the same.
func TestSlotPipelineFail_NotRunning(t *testing.T) {
	fakeYontrack(t, `{"data": {"failSlotPipeline": {"failStatus": {"ok": false, "message": "Only a running deployment can be marked as failed."}, "errors": []}}}`)
	cmd := cmdWithArgs(t, slotPipelineFailCmd, "--pipeline", "p1")

	assert.EqualError(t, cmd.RunE(cmd, nil), "Only a running deployment can be marked as failed.")
}

func TestSlotPipelineFail_Error(t *testing.T) {
	fakeYontrack(t, `{"data": {"failSlotPipeline": {"failStatus": null, "errors": [{"message": "Date/time cannot be in the future."}]}}}`)
	cmd := cmdWithArgs(t, slotPipelineFailCmd, "--pipeline", "p1")

	assert.EqualError(t, cmd.RunE(cmd, nil), "1) Date/time cannot be in the future.\n")
}

// No status and no error means nothing was done, whatever the HTTP status said.
func TestSlotPipelineFail_NoStatus(t *testing.T) {
	fakeYontrack(t, `{"data": {"failSlotPipeline": {"failStatus": null, "errors": []}}}`)
	cmd := cmdWithArgs(t, slotPipelineFailCmd, "--pipeline", "p1")

	assert.EqualError(t, cmd.RunE(cmd, nil), "the pipeline was not marked as failed, and no error was reported either")
}

// After 'slot pipeline start --output env', the pipeline needs no flag. Without
// --message or --date, nothing is sent for them: Yontrack takes its default
// message and the current time, and an empty date/time would not parse.
func TestSlotPipelineFail_Defaults(t *testing.T) {
	t.Setenv("YONTRACK_PIPELINE_ID", "p1")
	request := fakeYontrack(t, `{"data": {"failSlotPipeline": {"failStatus": {"ok": true, "message": "Deployment failed"}, "errors": []}}}`)
	cmd := cmdWithArgs(t, slotPipelineFailCmd)

	require.NoError(t, cmd.RunE(cmd, nil))

	assert.Equal(t, map[string]interface{}{"pipelineId": "p1"}, slotPipelineFailInput(t, request))
}

// A refusal without a reason still fails, and says what was refused.
func TestSlotPipelineFail_RefusedWithoutMessage(t *testing.T) {
	fakeYontrack(t, `{"data": {"failSlotPipeline": {"failStatus": {"ok": null, "message": ""}, "errors": []}}}`)
	cmd := cmdWithArgs(t, slotPipelineFailCmd, "--pipeline", "p1")

	assert.EqualError(t, cmd.RunE(cmd, nil), "the pipeline was not marked as failed")
}
