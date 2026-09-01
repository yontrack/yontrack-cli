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

// decodeResponse parses a GraphQL `data` payload the way the command receives it.
func decodeResponse(t *testing.T, body string) interface{} {
	var data interface{}
	assert.NoError(t, json.Unmarshal([]byte(body), &data))
	return data
}

func TestCollectUserErrors_None(t *testing.T) {
	data := decodeResponse(t, `{"startSlotPipeline": {"pipeline": {"id": "p1"}, "errors": []}}`)

	assert.Empty(t, collectUserErrors(data))
}

// A refused mutation arrives as ordinary data with a 200, so this is the only
// thing standing between it and an exit code of 0.
func TestCollectUserErrors_Refused(t *testing.T) {
	data := decodeResponse(t, `{"startSlotPipeline": {"pipeline": null, "errors": [{"message": "Build is not eligible"}]}}`)

	assert.Equal(t, []string{"startSlotPipeline: Build is not eligible"}, collectUserErrors(data))
}

func TestCollectUserErrors_NullErrors(t *testing.T) {
	data := decodeResponse(t, `{"startSlotPipeline": {"pipeline": {"id": "p1"}, "errors": null}}`)

	assert.Empty(t, collectUserErrors(data))
}

// Several mutations can be sent in one request; each one's messages are
// attributed to the field that carried them, in a stable order.
func TestCollectUserErrors_MultiplePayloads(t *testing.T) {
	data := decodeResponse(t, `{
		"zebra": {"errors": [{"message": "last alphabetically"}]},
		"alpha": {"errors": [{"message": "first"}, {"message": "second"}]}
	}`)

	assert.Equal(t, []string{
		"alpha: first",
		"alpha: second",
		"zebra: last alphabetically",
	}, collectUserErrors(data))
}

// A query response carries no payloads at all and must not be mistaken for a
// failure.
func TestCollectUserErrors_QueryResponse(t *testing.T) {
	data := decodeResponse(t, `{"projects": [{"id": "1", "name": "ontrack"}]}`)

	assert.Empty(t, collectUserErrors(data))
}

func TestCollectUserErrors_NotAnObject(t *testing.T) {
	assert.Empty(t, collectUserErrors(nil))
	assert.Empty(t, collectUserErrors(decodeResponse(t, `[1, 2]`)))
}

// An `errors` field that is not the Payload shape must be ignored rather than
// crash the command: not every field called "errors" belongs to a mutation.
func TestCollectUserErrors_UnexpectedShapes(t *testing.T) {
	assert.Empty(t, collectUserErrors(decodeResponse(t, `{"thing": {"errors": "not a list"}}`)))
	assert.Empty(t, collectUserErrors(decodeResponse(t, `{"thing": {"errors": [{"noMessage": 1}]}}`)))
	assert.Empty(t, collectUserErrors(decodeResponse(t, `{"thing": {"errors": ["plain string"]}}`)))
}
