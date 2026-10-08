package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The exports shared with Yontrack, copied with the audittrail tests
func sharedExport(name string) string {
	return filepath.Join("..", "audittrail", "testdata", "test-vectors", "exports", name)
}

// runAuditTrailVerify runs 'audit-trail verify' on a file, and returns its
// report and its error.
func runAuditTrailVerify(t *testing.T, file string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	auditTrailVerifyCmd.SetOut(&out)
	t.Cleanup(func() { auditTrailVerifyCmd.SetOut(nil) })
	err := auditTrailVerifyCmd.RunE(auditTrailVerifyCmd, []string{file})
	return out.String(), err
}

func TestAuditTrailVerifyIntact(t *testing.T) {
	report, err := runAuditTrailVerify(t, sharedExport("01-build-created.json"))

	require.NoError(t, err)
	assert.Equal(t, `Trail of build payments / release-2.4 / 2.4.7 (ID 1042), exported at 2026-10-02T09:00:00.000Z
Entries:      3
Keys:         06e3fd8fda29bb60 (Ed25519)
Start:        complete, opened by build.created
Chain:        intact
Endorsements: valid
The trail is intact.
`, report)
}

// A tampered entry fails the command, after a report naming it.
func TestAuditTrailVerifyTamperedEntry(t *testing.T) {
	report, err := runAuditTrailVerify(t, sharedExport("02-tampered-payload.json"))

	assert.EqualError(t, err, "the trail is broken: its chain is broken at seq 2")
	assert.Equal(t, `Trail of build payments / release-2.4 / 2.4.7 (ID 1042), exported at 2026-10-02T09:00:00.000Z
Entries:      3
Keys:         06e3fd8fda29bb60 (Ed25519)
Start:        complete, opened by build.created
Chain:        broken at seq 2, verified up to seq 1
Endorsements: valid
Problems:
  seq 2  HASH  The hash of the entry is not the hash of its content.
`, report)
}

// writeEditedExport writes a shared export after replacing a part of it.
func writeEditedExport(t *testing.T, name, old, new string) string {
	t.Helper()
	content, err := os.ReadFile(sharedExport(name))
	require.NoError(t, err)
	require.Contains(t, string(content), old)
	path := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(path, []byte(strings.Replace(string(content), old, new, 1)), 0o600))
	return path
}

// The link of seq 3 points at another entry than seq 2.
func TestAuditTrailVerifyBrokenLink(t *testing.T) {
	file := writeEditedExport(t, "01-build-created.json",
		`"prevHash": "8a78e74944dfb4d80786ff009c402384a693b357769811fecc156ba78eecf7d5"`,
		`"prevHash": "4b320b3f1628780eaf5adbc9153975aa05eeb25fd082f97b111588499a6550a6"`)

	report, err := runAuditTrailVerify(t, file)

	assert.EqualError(t, err, "the trail is broken: its chain is broken at seq 3")
	assert.Contains(t, report, `Chain:        broken at seq 3, verified up to seq 2
Endorsements: valid
Problems:
  seq 3  HASH           The hash of the entry is not the hash of its content.
  seq 3  PREVIOUS_HASH  The previous hash of the entry is not the hash of the entry before it.
`)
}

// Seq 3 carries the signature of seq 2.
func TestAuditTrailVerifyBadEndorsement(t *testing.T) {
	file := writeEditedExport(t, "01-build-created.json",
		`"signature": "3YT8KN1uoDnSyjjtzPLMZda64XERC++yk8jf85BsnTgVs3+Q9EVXs7JQwW/D2IAXzjhD2irjfTE0GNaNrPMsBQ=="`,
		`"signature": "jy/gS3HJFRtrw0eZcJ491aAfKgOkjtB02TOxTb9ZTCN2a3AkCk4SQA/Uu/PLDgBUSElyn63PqkhniVYEErHxBQ=="`)

	report, err := runAuditTrailVerify(t, file)

	assert.EqualError(t, err, "the trail is broken: an endorsement is invalid at seq 3")
	assert.Contains(t, report, `Chain:        intact
Endorsements: invalid at seq 3
Problems:
  seq 3  ENDORSEMENT  The endorsement by key 06e3fd8fda29bb60 is not the signature of the hash of the entry.
`)
}

// A trail opened on an older build, written while the instance key was not
// provisioned, is reported as such - and is not broken.
func TestAuditTrailVerifyPartialUnendorsed(t *testing.T) {
	file := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(file, []byte(`{
		"exportVersion": 1,
		"exportedAt": "2026-10-02T09:30:00.000Z",
		"build": {"id": 1042, "project": "payments", "branch": "release-2.4", "name": "2.4.7"},
		"keys": [],
		"entries": [{
			"seq": 1, "schemaVersion": 1, "type": "trail.opened", "time": "2026-10-02T09:00:00.123Z",
			"actor": {"account": "alice", "via": "ui"}, "prevHash": null,
			"payload": {"build": {"id": 1042, "project": "payments", "branch": "release-2.4", "name": "2.4.7"}, "buildCreatedAt": "2026-09-30T17:42:08.500Z", "partial": true},
			"hash": "27a668764f0cabb7b03da21eac285ac3d364be2c8c6695e463be05e525e455fc",
			"endorsements": []
		}]
	}`), 0o600))

	report, err := runAuditTrailVerify(t, file)

	require.NoError(t, err)
	assert.Equal(t, `Trail of build payments / release-2.4 / 2.4.7 (ID 1042), exported at 2026-10-02T09:30:00.000Z
Entries:      1
Keys:         none
Start:        partial, the build predates its trail, which starts at seq 1 on 2026-10-02T09:00:00.123Z
Chain:        intact
Endorsements: valid
Unendorsed:   from seq 1, written while the instance key was not provisioned
The trail is intact.
`, report)
}

func TestAuditTrailVerifyNotAnExport(t *testing.T) {
	file := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"exportVersion": 2}`), 0o600))

	_, err := runAuditTrailVerify(t, file)

	assert.EqualError(t, err, "export version 2 is not supported: this CLI verifies export version 1")
}

func TestAuditTrailVerifyMissingFile(t *testing.T) {
	_, err := runAuditTrailVerify(t, filepath.Join(t.TempDir(), "missing.json"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read the export")
}
