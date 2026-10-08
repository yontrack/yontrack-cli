package audittrail

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// vectorFile is a file of the shared test vectors: a trail whose entries give
// their envelope, its canonical form and its hash.
type vectorFile struct {
	Description string
	Entries     []struct {
		Envelope  Envelope
		Canonical string
		Hash      string
	}
}

func readVectors(t *testing.T, name string) vectorFile {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", "test-vectors", name))
	require.NoError(t, err)
	var file vectorFile
	require.NoError(t, json.Unmarshal(content, &file))
	require.NotEmpty(t, file.Entries)
	return file
}

// Every envelope of the shared vectors hashes, through its canonical form, to
// the hash Yontrack computed for it.
func TestHashOfTheSharedVectors(t *testing.T) {
	for _, name := range []string{"01-build-created.json", "02-trail-opened.json", "03-canonical-forms.json"} {
		t.Run(name, func(t *testing.T) {
			for _, entry := range readVectors(t, name).Entries {
				canonical, err := Canonical(entry.Envelope)
				require.NoError(t, err)
				assert.Equal(t, entry.Canonical, string(canonical), "canonical form of seq %d", entry.Envelope.Seq)

				hash, err := Hash(entry.Envelope)
				require.NoError(t, err)
				assert.Equal(t, entry.Hash, hash, "hash of seq %d", entry.Envelope.Seq)
			}
		})
	}
}

// Outside the subset of JSON the trail accepts, there is no hash: Yontrack
// never writes such an entry, so one is found only in an edited export.
func TestHashRefusesWhatYontrackNeverHashes(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		error   string
	}{
		{"decimal", `{"value": 1.5}`, "canonical JSON accepts only integers within ±(2^53 - 1), found 1.5 at /payload/value"},
		{"integral decimal", `{"value": 1.0}`, "canonical JSON accepts only integers within ±(2^53 - 1), found 1.0 at /payload/value"},
		{"exponent", `{"list": [1e3]}`, "canonical JSON accepts only integers within ±(2^53 - 1), found 1e3 at /payload/list/0"},
		{"too large", `{"value": 9007199254740992}`, "canonical JSON accepts only integers within ±(2^53 - 1), found 9007199254740992 at /payload/value"},
		{"too small", `{"value": -9007199254740992}`, "canonical JSON accepts only integers within ±(2^53 - 1), found -9007199254740992 at /payload/value"},
		{"lone high surrogate", `{"value": "\ud800A"}`, "canonical JSON accepts only well-formed Unicode strings, found a lone surrogate in the payload"},
		{"high surrogate before another escape", `{"value": "\ud800\u0041"}`, "canonical JSON accepts only well-formed Unicode strings, found a lone surrogate in the payload"},
		{"lone low surrogate", `{"\udc00": "x"}`, "canonical JSON accepts only well-formed Unicode strings, found a lone surrogate in the payload"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			envelope := Envelope{
				SchemaVersion: 1,
				Seq:           1,
				Type:          "build.created",
				Time:          "2026-10-02T08:15:30.000Z",
				Actor:         json.RawMessage(`{"account": "alice", "via": "ui"}`),
				Payload:       json.RawMessage(c.payload),
			}

			_, err := Hash(envelope)

			assert.EqualError(t, err, c.error)
		})
	}
}

// An escaped backslash followed by a "u" is not a \u escape.
func TestHashOfAnEscapedBackslashBeforeU(t *testing.T) {
	envelope := Envelope{
		SchemaVersion: 1,
		Seq:           1,
		Type:          "build.created",
		Time:          "2026-10-02T08:15:30.000Z",
		Actor:         json.RawMessage(`{"account": "alice", "via": "ui"}`),
		Payload:       json.RawMessage(`{"path": "C:\\ud800"}`),
	}

	canonical, err := Canonical(envelope)

	require.NoError(t, err)
	assert.Contains(t, string(canonical), `"payload":{"path":"C:\\ud800"}`)
}
