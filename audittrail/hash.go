// Package audittrail verifies offline the JSON export of the audit trail of a
// Yontrack build: its hash chain and the endorsements of its entries by the
// instance key, with nothing but the content of the export.
package audittrail

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

// SchemaVersion is the only schema version of the entries this CLI hashes:
// hash format v1.
const SchemaVersion = 1

// Envelope is what is hashed of an entry: everything but its hash.
type Envelope struct {
	SchemaVersion int             `json:"schemaVersion"`
	Seq           int             `json:"seq"`
	Type          string          `json:"type"`
	Time          string          `json:"time"`
	Actor         json.RawMessage `json:"actor"`
	PrevHash      *string         `json:"prevHash"`
	Payload       json.RawMessage `json:"payload"`
}

// Hash is the hash of an envelope with hash format v1: the SHA-256, in
// lowercase hexadecimal, of the UTF-8 bytes of its canonical form.
func Hash(envelope Envelope) (string, error) {
	canonical, err := Canonical(envelope)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

// Canonical is the RFC 8785 canonical form of an envelope, the bytes which are
// hashed.
//
// Yontrack accepts only a subset of JSON in an entry - strings, objects,
// arrays, booleans, null and integers within ±(2^53 - 1) - and so does this
// function: anything else would hash here while Yontrack refuses to hash it.
// Within that subset, the canonical form is the one of the reference
// implementation of RFC 8785.
func Canonical(envelope Envelope) ([]byte, error) {
	if envelope.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("hash format v1 only hashes the entries of schema version %d, not %d", SchemaVersion, envelope.SchemaVersion)
	}
	if err := checkSubset("actor", envelope.Actor); err != nil {
		return nil, err
	}
	if err := checkSubset("payload", envelope.Payload); err != nil {
		return nil, err
	}
	text, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	return jsoncanonicalizer.Transform(text)
}

// Largest integer of the subset, 2^53 - 1
var maxInteger = new(big.Int).SetInt64(1<<53 - 1)

var integerRegex = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

// checkSubset checks that a member of the envelope holds only what Yontrack
// hashes.
func checkSubset(name string, raw json.RawMessage) error {
	if raw == nil {
		return nil
	}
	if err := checkSurrogates(raw); err != nil {
		return fmt.Errorf("%s in the %s", err, name)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("the %s is not JSON: %w", name, err)
	}
	return checkValue(value, "/"+name)
}

func checkValue(value interface{}, path string) error {
	switch v := value.(type) {
	case json.Number:
		text := v.String()
		integer, ok := new(big.Int).SetString(text, 10)
		if !integerRegex.MatchString(text) || !ok || new(big.Int).Abs(integer).Cmp(maxInteger) > 0 {
			return fmt.Errorf("canonical JSON accepts only integers within ±(2^53 - 1), found %s at %s", text, path)
		}
	case []interface{}:
		for index, item := range v {
			if err := checkValue(item, path+"/"+strconv.Itoa(index)); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		names := make([]string, 0, len(v))
		for name := range v {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if err := checkValue(v[name], path+"/"+escapePointer(name)); err != nil {
				return err
			}
		}
	}
	return nil
}

// escapePointer escapes a property name as a JSON Pointer token (RFC 6901).
func escapePointer(name string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(name)
}

var errLoneSurrogate = errors.New("canonical JSON accepts only well-formed Unicode strings, found a lone surrogate")

// checkSurrogates looks for a \u escape of a lone surrogate in a JSON text.
// Decoding cannot tell: Go and the reference canonicalizer both turn some of
// them into U+FFFD, which would let an edited string hash as the original.
func checkSurrogates(raw []byte) error {
	inString := false
	pendingHigh := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if !inString {
			if c == '"' {
				inString = true
			}
			continue
		}
		if c == '\\' && i+1 < len(raw) && raw[i+1] == 'u' && i+5 < len(raw) {
			unit, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
			if err != nil {
				return err
			}
			r := rune(unit)
			i += 5
			switch {
			case utf16.IsSurrogate(r) && r < 0xdc00: // high
				if pendingHigh {
					return errLoneSurrogate
				}
				pendingHigh = true
			case utf16.IsSurrogate(r): // low
				if !pendingHigh {
					return errLoneSurrogate
				}
				pendingHigh = false
			default:
				if pendingHigh {
					return errLoneSurrogate
				}
			}
			continue
		}
		if pendingHigh {
			return errLoneSurrogate
		}
		switch c {
		case '\\':
			i++ // the escaped character
		case '"':
			inString = false
		}
	}
	return nil
}
