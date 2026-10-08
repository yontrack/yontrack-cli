package audittrail

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

// ExportVersion is the only version of the shape of an export this CLI reads.
const ExportVersion = 1

// Export is the JSON export of the trail of a build, as Yontrack writes it:
// self-sufficient, it is verified with nothing but its own content.
type Export struct {
	ExportVersion int     `json:"exportVersion"`
	ExportedAt    string  `json:"exportedAt"`
	Build         Build   `json:"build"`
	Keys          []Key   `json:"keys"`
	Entries       []Entry `json:"entries"`
}

// Build is the build of an exported trail, which its first entry names.
type Build struct {
	Id      int    `json:"id"`
	Project string `json:"project"`
	Branch  string `json:"branch"`
	Name    string `json:"name"`
}

// Key is a public key of the instance, with which its endorsements are
// verified.
type Key struct {
	KeyId     string `json:"keyId"`
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"publicKey"`
}

// Entry is an exported entry: its envelope, as hashed, its hash and its
// endorsements.
type Entry struct {
	Envelope
	Hash         string        `json:"hash"`
	Endorsements []Endorsement `json:"endorsements"`
}

// Endorsement is the Ed25519 signature of the hash of an entry by a key of
// the instance.
type Endorsement struct {
	KeyId     string `json:"keyId"`
	Signature string `json:"signature"`
	Time      string `json:"time"`
}

// ReadExport reads a JSON export.
func ReadExport(content []byte) (*Export, error) {
	var export Export
	if err := json.Unmarshal(content, &export); err != nil {
		return nil, fmt.Errorf("not an export of an audit trail: %w", err)
	}
	if export.ExportVersion != ExportVersion {
		return nil, fmt.Errorf("export version %d is not supported: this CLI verifies export version %d", export.ExportVersion, ExportVersion)
	}
	return &export, nil
}

// ProblemType is a check of the verification. All but ENDORSEMENT and
// UNKNOWN_KEY are checks of the chain.
type ProblemType string

const (
	// ProblemSeq - the entry is not at the position its seq gives
	ProblemSeq ProblemType = "SEQ"
	// ProblemSchemaVersion - the schema version of the entry is not supported
	ProblemSchemaVersion ProblemType = "SCHEMA_VERSION"
	// ProblemHash - the hash of the entry is not the hash of its content
	ProblemHash ProblemType = "HASH"
	// ProblemPreviousHash - the entry is not chained to the one before it
	ProblemPreviousHash ProblemType = "PREVIOUS_HASH"
	// ProblemFirstEntry - the first entry is neither build.created nor trail.opened
	ProblemFirstEntry ProblemType = "FIRST_ENTRY"
	// ProblemBuild - the first entry names another build
	ProblemBuild ProblemType = "BUILD"
	// ProblemEndorsement - an endorsement is not the signature of the hash by its key
	ProblemEndorsement ProblemType = "ENDORSEMENT"
	// ProblemUnknownKey - an endorsement names a key which is not in the export
	ProblemUnknownKey ProblemType = "UNKNOWN_KEY"
)

// Chain tells whether a failure of this check breaks the chain.
func (t ProblemType) Chain() bool {
	return t != ProblemEndorsement && t != ProblemUnknownKey
}

// Problem is a check which failed, at a position of the trail, from 1.
type Problem struct {
	Seq     int
	Type    ProblemType
	Message string
}

// Verification is the result of the verification of a trail. A position is 0
// when there is none.
type Verification struct {
	// ChainIntact - every hash recomputes, every entry is chained to the one
	// before it, and the seqs run from 1 without gap
	ChainIntact bool
	// FirstBrokenSeq - first entry failing a check of the chain
	FirstBrokenSeq int
	// EndorsementsValid - every endorsement is the signature of the hash of
	// its entry by a key of the export
	EndorsementsValid bool
	// FirstInvalidEndorsementSeq - first entry with an invalid endorsement
	FirstInvalidEndorsementSeq int
	// Partial - the trail opens with trail.opened: its build predates it
	Partial bool
	// UnendorsedFromSeq - first entry after the last endorsed one
	UnendorsedFromSeq int
	// Problems - every check which failed, by position
	Problems []Problem
}

// Broken tells whether the trail cannot be trusted: its chain is broken, or an
// endorsement is invalid. A partial trail, or an unendorsed tail, is not
// broken.
func (v Verification) Broken() bool {
	return !v.ChainIntact || !v.EndorsementsValid
}

const (
	typeBuildCreated = "build.created"
	typeTrailOpened  = "trail.opened"
)

// Verify verifies an exported trail with its own keys, against the build it
// names, the way Yontrack verifies it. For the entry at position n, from 1:
//
//  1. its seq is n;
//  2. its schema version is supported: 1;
//  3. the hash of its envelope is its hash;
//  4. its previous hash is null for n = 1, the hash of the entry n - 1 otherwise;
//  5. for n = 1, its type is build.created or trail.opened, and it names the
//     build of the export;
//  6. each of its endorsements names a key of the export, and is the Ed25519
//     signature of its hash by that key.
//
// Checks 1 to 5 are the chain. An entry without endorsement is not invalid: it
// was written while the instance key was not provisioned.
func Verify(export *Export) Verification {
	keys := publicKeys(export.Keys)
	var problems []Problem
	for index, entry := range export.Entries {
		position := index + 1
		problem := func(problemType ProblemType, format string, args ...interface{}) {
			problems = append(problems, Problem{Seq: position, Type: problemType, Message: fmt.Sprintf(format, args...)})
		}

		if entry.Seq != position {
			problem(ProblemSeq, "Seq %d found at position %d.", entry.Seq, position)
		}
		if entry.SchemaVersion != SchemaVersion {
			problem(ProblemSchemaVersion, "Schema version %d is not supported.", entry.SchemaVersion)
		} else if hash, err := Hash(entry.Envelope); err != nil {
			problem(ProblemHash, "The content of the entry cannot be hashed: %s.", err)
		} else if hash != entry.Hash {
			problem(ProblemHash, "The hash of the entry is not the hash of its content.")
		}
		if index == 0 {
			if entry.PrevHash != nil {
				problem(ProblemPreviousHash, "The first entry has a previous hash.")
			}
		} else if entry.PrevHash == nil || *entry.PrevHash != export.Entries[index-1].Hash {
			problem(ProblemPreviousHash, "The previous hash of the entry is not the hash of the entry before it.")
		}
		if index == 0 {
			if entry.Type != typeBuildCreated && entry.Type != typeTrailOpened {
				problem(ProblemFirstEntry, "The first entry is %s, neither %s nor %s.", entry.Type, typeBuildCreated, typeTrailOpened)
			}
			if !namesBuild(entry.Payload, export.Build.Id) {
				problem(ProblemBuild, "The first entry does not name build %d.", export.Build.Id)
			}
		}
		for _, endorsement := range entry.Endorsements {
			key, known := keys[endorsement.KeyId]
			if !known {
				problem(ProblemUnknownKey, "Endorsed by key %s, which is not a public key of the instance.", endorsement.KeyId)
			} else if !endorses(key, entry.Hash, endorsement.Signature) {
				problem(ProblemEndorsement, "The endorsement by key %s is not the signature of the hash of the entry.", endorsement.KeyId)
			}
		}
	}

	sort.SliceStable(problems, func(i, j int) bool { return problems[i].Seq < problems[j].Seq })
	verification := Verification{Problems: problems}
	for _, problem := range problems {
		if problem.Type.Chain() && verification.FirstBrokenSeq == 0 {
			verification.FirstBrokenSeq = problem.Seq
		}
		if !problem.Type.Chain() && verification.FirstInvalidEndorsementSeq == 0 {
			verification.FirstInvalidEndorsementSeq = problem.Seq
		}
	}
	verification.ChainIntact = verification.FirstBrokenSeq == 0
	verification.EndorsementsValid = verification.FirstInvalidEndorsementSeq == 0
	verification.Partial = len(export.Entries) > 0 && export.Entries[0].Type == typeTrailOpened
	lastEndorsed := 0
	for index, entry := range export.Entries {
		if len(entry.Endorsements) > 0 {
			lastEndorsed = index + 1
		}
	}
	if lastEndorsed < len(export.Entries) {
		verification.UnendorsedFromSeq = lastEndorsed + 1
	}
	return verification
}

// namesBuild tells whether the payload of the first entry names the build:
// its build.id is the ID of the build.
func namesBuild(payload json.RawMessage, buildId int) bool {
	var content struct {
		Build struct {
			// Raw, so that only the integer itself names the build - not a
			// string holding it, nor a decimal
			Id json.RawMessage `json:"id"`
		} `json:"build"`
	}
	if err := json.Unmarshal(payload, &content); err != nil {
		return false
	}
	return string(bytes.TrimSpace(content.Build.Id)) == strconv.Itoa(buildId)
}

var hashRegex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// endorses tells whether a signature, in base64, is the Ed25519 signature of
// the 32 bytes of a hash by a key.
func endorses(key ed25519.PublicKey, hash, signature string) bool {
	if !hashRegex.MatchString(hash) {
		return false
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	hashBytes, _ := hex.DecodeString(hash)
	return ed25519.Verify(key, hashBytes, signatureBytes)
}

// publicKeys reads the keys of an export by ID. A key which cannot be read,
// which is not Ed25519, or whose ID is not the ID of its public key, is
// ignored: what it endorsed is reported as endorsed by an unknown key.
func publicKeys(keys []Key) map[string]ed25519.PublicKey {
	result := make(map[string]ed25519.PublicKey)
	for _, key := range keys {
		if key.Algorithm != "Ed25519" {
			continue
		}
		block, _ := pem.Decode([]byte(key.PublicKey))
		if block == nil || block.Type != "PUBLIC KEY" {
			continue
		}
		parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			continue
		}
		publicKey, ok := parsed.(ed25519.PublicKey)
		if !ok || KeyId(block.Bytes) != key.KeyId {
			continue
		}
		result[key.KeyId] = publicKey
	}
	return result
}

// KeyId is the ID of a public key: the first 16 lowercase hexadecimal
// characters of the SHA-256 of its DER SubjectPublicKeyInfo.
func KeyId(der []byte) string {
	digest := sha256.Sum256(der)
	return hex.EncodeToString(digest[:])[:16]
}
