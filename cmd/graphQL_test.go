package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildVariables_VarsOnly(t *testing.T) {
	variables, err := buildVariables("", []string{"name=ontrack", "branch=main"})

	assert.NoError(t, err)
	assert.Equal(t, "ontrack", variables["name"])
	assert.Equal(t, "main", variables["branch"])
}

// The point of --vars-json: an Int variable has to reach the server as a number.
// Sent as a string, the server rejects it.
func TestBuildVariables_JsonKeepsTypes(t *testing.T) {
	variables, err := buildVariables(`{"buildId": 11404, "dryRun": false, "slotId": "abc"}`, nil)

	assert.NoError(t, err)
	assert.Equal(t, "abc", variables["slotId"])
	assert.Equal(t, false, variables["dryRun"])

	// Re-marshalled the way the client sends it: a bare number, not a quoted one.
	body, err := json.Marshal(variables["buildId"])
	assert.NoError(t, err)
	assert.Equal(t, "11404", string(body))
}

// Numbers are decoded with UseNumber, so a value too large for an exact float64
// still round-trips verbatim rather than being silently rounded.
func TestBuildVariables_LargeNumberPrecision(t *testing.T) {
	variables, err := buildVariables(`{"big": 9007199254740993}`, nil)

	assert.NoError(t, err)
	body, err := json.Marshal(variables["big"])
	assert.NoError(t, err)
	assert.Equal(t, "9007199254740993", string(body))
}

func TestBuildVariables_NestedObject(t *testing.T) {
	variables, err := buildVariables(`{"input": {"slotId": "abc", "buildId": 11404}}`, nil)

	assert.NoError(t, err)
	body, err := json.Marshal(variables["input"])
	assert.NoError(t, err)
	assert.JSONEq(t, `{"slotId":"abc","buildId":11404}`, string(body))
}

// --var is applied last so a single string can be overridden without rewriting
// the whole JSON object.
func TestBuildVariables_VarWinsOverJson(t *testing.T) {
	variables, err := buildVariables(`{"name": "from-json", "buildId": 1}`, []string{"name=from-var"})

	assert.NoError(t, err)
	assert.Equal(t, "from-var", variables["name"])
	assert.NotNil(t, variables["buildId"])
}

func TestBuildVariables_InvalidJson(t *testing.T) {
	_, err := buildVariables(`{not json`, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not valid JSON")
}

// The variables of a GraphQL request are an object by definition; anything else
// is a mistake worth reporting rather than coercing.
func TestBuildVariables_NotAnObject(t *testing.T) {
	_, err := buildVariables(`[1, 2, 3]`, nil)

	assert.EqualError(t, err, "--vars-json must be a JSON object")

	_, err = buildVariables(`"just a string"`, nil)
	assert.EqualError(t, err, "--vars-json must be a JSON object")
}

func TestBuildVariables_Empty(t *testing.T) {
	variables, err := buildVariables("", nil)

	assert.NoError(t, err)
	assert.Empty(t, variables)
}

func TestParseVar(t *testing.T) {
	name, value, err := parseVar("name=ontrack")
	assert.NoError(t, err)
	assert.Equal(t, "name", name)
	assert.Equal(t, "ontrack", value)

	// An empty value is legitimate.
	name, value, err = parseVar("name=")
	assert.NoError(t, err)
	assert.Equal(t, "name", name)
	assert.Equal(t, "", value)

	_, _, err = parseVar("noequalsign")
	assert.Error(t, err)
}
