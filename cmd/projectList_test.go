package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Project.id is an `ID!` in the schema, so the server sends it as a JSON string
// and never as a number. Decoding it into an int fails the whole response.
func TestProjectListDecodesStringIDs(t *testing.T) {
	payload := `{"projects":[{"id":"164","name":"accounting"},{"id":"165","name":"auth0"}]}`

	data := new(projectListResponse)
	assert.NoError(t, json.Unmarshal([]byte(payload), data))

	assert.Len(t, data.Projects, 2)
	assert.Equal(t, "164", data.Projects[0].ID)
	assert.Equal(t, "accounting", data.Projects[0].Name)
	assert.Equal(t, "165", data.Projects[1].ID)
	assert.Equal(t, "auth0", data.Projects[1].Name)
}
