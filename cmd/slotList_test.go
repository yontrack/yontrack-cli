package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const slotListResponse = `{"data": {"environments": [
	{"name": "production", "order": 30, "slots": [
		{"id": "slot-3", "qualifier": "", "lastDeployedPipeline": null}
	]},
	{"name": "staging", "order": 20, "slots": [
		{"id": "slot-2", "qualifier": "", "lastDeployedPipeline": {"build": {"name": "45", "displayName": "1.2.3"}}},
		{"id": "slot-4", "qualifier": "eu", "lastDeployedPipeline": {"build": {"name": "44", "displayName": "1.2.2"}}}
	]},
	{"name": "dev", "order": 10, "slots": [
		{"id": "slot-1", "qualifier": "", "lastDeployedPipeline": {"build": {"name": "46", "displayName": "1.2.4"}}}
	]}
]}}`

// One row per slot, in environment order, and a dash for a slot which never
// completed a deployment.
func TestSlotList(t *testing.T) {
	requests := fakeYontrackRoutes(t, map[string]string{"environments(": slotListResponse})

	err, out := runWithOutput(t, slotListCmd, "--project", "my-project")

	require.NoError(t, err)
	// Columns: environment, qualifier (empty for the default slot), slot ID,
	// deployed build
	assert.Equal(t, ""+
		"dev             slot-1  1.2.4\n"+
		"staging         slot-2  1.2.3\n"+
		"staging     eu  slot-4  1.2.2\n"+
		"production      slot-3  —\n", out)
	require.Len(t, *requests, 1)
	assert.Equal(t, map[string]interface{}{"project": "my-project"}, (*requests)[0].Variables)
	assert.Contains(t, (*requests)[0].Query, "environments(filter: {projects: [$project]})")
	assert.Contains(t, (*requests)[0].Query, "slots(projects: [$project])")
}

func TestSlotListJSON(t *testing.T) {
	fakeYontrackRoutes(t, map[string]string{"environments(": slotListResponse})

	err, out := runWithOutput(t, slotListCmd, "--project", "my-project", "--output", "json")

	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"name": "dev", "order": 10, "slots": [
			{"id": "slot-1", "qualifier": "", "lastDeployedPipeline": {"build": {"name": "46", "displayName": "1.2.4"}}}
		]},
		{"name": "staging", "order": 20, "slots": [
			{"id": "slot-2", "qualifier": "", "lastDeployedPipeline": {"build": {"name": "45", "displayName": "1.2.3"}}},
			{"id": "slot-4", "qualifier": "eu", "lastDeployedPipeline": {"build": {"name": "44", "displayName": "1.2.2"}}}
		]},
		{"name": "production", "order": 30, "slots": [
			{"id": "slot-3", "qualifier": "", "lastDeployedPipeline": null}
		]}
	]`, out)
}

func TestSlotListProjectRequired(t *testing.T) {
	t.Setenv("YONTRACK_PROJECT_NAME", "")
	fakeYontrackRoutes(t, map[string]string{})

	err, _ := runWithOutput(t, slotListCmd)

	assert.EqualError(t, err, "project is required (use --project flag or YONTRACK_PROJECT_NAME environment variable)")
}
