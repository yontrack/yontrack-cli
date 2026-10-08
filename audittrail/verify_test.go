package audittrail

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readExport(t *testing.T, name string) *Export {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", "test-vectors", "exports", name))
	require.NoError(t, err)
	export, err := ReadExport(content)
	require.NoError(t, err)
	return export
}

// problemsOf lists the failed checks as seq and type, which is what they mean.
func problemsOf(verification Verification) []string {
	var list []string
	for _, problem := range verification.Problems {
		list = append(list, string(rune('0'+problem.Seq))+" "+string(problem.Type))
	}
	return list
}

func TestVerifyIntactExport(t *testing.T) {
	verification := Verify(readExport(t, "01-build-created.json"))

	assert.True(t, verification.ChainIntact)
	assert.Equal(t, 0, verification.FirstBrokenSeq)
	assert.True(t, verification.EndorsementsValid)
	assert.Equal(t, 0, verification.FirstInvalidEndorsementSeq)
	assert.False(t, verification.Partial)
	assert.Equal(t, 0, verification.UnendorsedFromSeq)
	assert.Empty(t, verification.Problems)
	assert.False(t, verification.Broken())
}

// The payload of seq 2 was edited, its hash left as it was: the hash no
// longer recomputes. Its endorsement still signs the stored hash.
func TestVerifyTamperedEntry(t *testing.T) {
	verification := Verify(readExport(t, "02-tampered-payload.json"))

	assert.False(t, verification.ChainIntact)
	assert.Equal(t, 2, verification.FirstBrokenSeq)
	assert.True(t, verification.EndorsementsValid)
	assert.Equal(t, []string{"2 HASH"}, problemsOf(verification))
	assert.True(t, verification.Broken())
}

// The link of seq 3 to seq 2 was changed.
func TestVerifyBrokenLink(t *testing.T) {
	export := readExport(t, "01-build-created.json")
	other := "0000000000000000000000000000000000000000000000000000000000000000"
	export.Entries[2].PrevHash = &other

	verification := Verify(export)

	assert.False(t, verification.ChainIntact)
	assert.Equal(t, 3, verification.FirstBrokenSeq)
	assert.Equal(t, []string{"3 HASH", "3 PREVIOUS_HASH"}, problemsOf(verification))
}

// Seq 2 was removed: seq 3 is found at position 2, chained to an entry which
// is not there anymore.
func TestVerifyRemovedEntry(t *testing.T) {
	export := readExport(t, "01-build-created.json")
	export.Entries = append(export.Entries[:1], export.Entries[2])

	verification := Verify(export)

	assert.Equal(t, 2, verification.FirstBrokenSeq)
	assert.Equal(t, []string{"2 SEQ", "2 PREVIOUS_HASH"}, problemsOf(verification))
}

// Seq 3 carries the signature of seq 2: its chain is intact, but it is not
// endorsed by the key it names.
func TestVerifyBadEndorsementSignature(t *testing.T) {
	export := readExport(t, "01-build-created.json")
	export.Entries[2].Endorsements[0].Signature = export.Entries[1].Endorsements[0].Signature

	verification := Verify(export)

	assert.True(t, verification.ChainIntact)
	assert.False(t, verification.EndorsementsValid)
	assert.Equal(t, 3, verification.FirstInvalidEndorsementSeq)
	assert.Equal(t, []string{"3 ENDORSEMENT"}, problemsOf(verification))
	assert.True(t, verification.Broken())
}

// A signature which is not even base64 is a bad endorsement, not a crash.
func TestVerifyGarbledEndorsementSignature(t *testing.T) {
	export := readExport(t, "01-build-created.json")
	export.Entries[0].Endorsements[0].Signature = "not base64!"

	verification := Verify(export)

	assert.Equal(t, []string{"1 ENDORSEMENT"}, problemsOf(verification))
}

func TestVerifyEndorsementByUnknownKey(t *testing.T) {
	export := readExport(t, "01-build-created.json")
	export.Entries[1].Endorsements[0].KeyId = "0123456789abcdef"

	verification := Verify(export)

	assert.True(t, verification.ChainIntact)
	assert.Equal(t, 2, verification.FirstInvalidEndorsementSeq)
	assert.Equal(t, []string{"2 UNKNOWN_KEY"}, problemsOf(verification))
}

// A key whose ID is not the ID of its public key is ignored: what it endorsed
// is endorsed by an unknown key.
func TestVerifyKeyWithAnotherId(t *testing.T) {
	export := readExport(t, "01-build-created.json")
	export.Keys[0].PublicKey = "-----BEGIN PUBLIC KEY-----\nMCowBQYDK2VwAyEAPUAXw+hDiVqStwqnTRt+vJyYLM8uxJaMwM1V8Sr0Zgw=\n-----END PUBLIC KEY-----\n"

	verification := Verify(export)

	assert.Equal(t, []string{"1 UNKNOWN_KEY", "2 UNKNOWN_KEY", "3 UNKNOWN_KEY"}, problemsOf(verification))
}

// The trail is bound to the build the export names.
func TestVerifyExportOfAnotherBuild(t *testing.T) {
	export := readExport(t, "01-build-created.json")
	export.Build.Id = 1043

	verification := Verify(export)

	assert.Equal(t, 1, verification.FirstBrokenSeq)
	assert.Equal(t, []string{"1 BUILD"}, problemsOf(verification))
}

// A trail opened on a build which predates it is partial, never broken; and
// entries written while the instance key was not provisioned are the
// unendorsed tail, not invalid endorsements.
func TestVerifyPartialUnendorsedTrail(t *testing.T) {
	export := &Export{
		ExportVersion: 1,
		Build:         Build{Id: 1042, Project: "payments", Branch: "release-2.4", Name: "2.4.7"},
	}
	for _, entry := range readVectors(t, "02-trail-opened.json").Entries {
		export.Entries = append(export.Entries, Entry{Envelope: entry.Envelope, Hash: entry.Hash})
	}

	verification := Verify(export)

	assert.True(t, verification.ChainIntact)
	assert.True(t, verification.EndorsementsValid)
	assert.True(t, verification.Partial)
	assert.Equal(t, 1, verification.UnendorsedFromSeq)
	assert.Empty(t, verification.Problems)
	assert.False(t, verification.Broken())
}

func TestVerifyUnsupportedSchemaVersion(t *testing.T) {
	export := readExport(t, "01-build-created.json")
	export.Entries[2].SchemaVersion = 2

	verification := Verify(export)

	assert.Equal(t, []string{"3 SCHEMA_VERSION"}, problemsOf(verification))
}

func TestReadExportOfAnotherVersion(t *testing.T) {
	_, err := ReadExport([]byte(`{"exportVersion": 2, "entries": []}`))

	assert.EqualError(t, err, "export version 2 is not supported: this CLI verifies export version 1")
}

func TestReadExportWhichIsNotAnExport(t *testing.T) {
	_, err := ReadExport([]byte(`[]`))

	assert.Error(t, err)
}

// What is read is what Yontrack exported: nothing is lost on the way.
func TestReadExportKeepsTheEntries(t *testing.T) {
	export := readExport(t, "01-build-created.json")

	assert.Equal(t, "06e3fd8fda29bb60", export.Keys[0].KeyId)
	require.Len(t, export.Entries, 3)
	assert.Equal(t, "link.added", export.Entries[1].Type)
	assert.JSONEq(t, `{"target": {"id": 977, "project": "ledger-core", "branch": "main", "name": "1.18.0"}, "qualifier": ""}`,
		string(export.Entries[1].Payload))
	var actor map[string]string
	require.NoError(t, json.Unmarshal(export.Entries[1].Actor, &actor))
	assert.Equal(t, "ci-demo", actor["tokenName"])
}
