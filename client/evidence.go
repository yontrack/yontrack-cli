package client

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	config "yontrack/config"
)

// EvidenceSource is where an evidence comes from, as the CLI claims it. Each
// field is optional.
type EvidenceSource struct {
	// Tool which produced the evidence, like trivy
	Tool string
	// Version of the tool
	Version string
	// Where the evidence was produced - a CI job, a report: an HTTP or HTTPS URL
	URL string
}

// EvidenceRefusedError is the answer of Yontrack when it does not attach an
// evidence.
type EvidenceRefusedError struct {
	// HTTP status of the answer
	Status int
	// Stable code of the refusal, like audit-trail.evidence.too-large - empty
	// when the refusal is not one of an evidence (no right to upload, say)
	Code string
	// Message of Yontrack
	Message string
}

func (e *EvidenceRefusedError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s (%s)", e.Message, e.Code)
	}
	return fmt.Sprintf("%s (HTTP %d)", e.Message, e.Status)
}

// UploadEvidence attaches a file as evidence to a validation run, through the
// multipart REST end point of Yontrack 6.0 - there is no GraphQL mutation for
// it.
//
// The SHA-256 of the file is sent as its external digest: Yontrack computes
// its own and refuses the evidence if they differ, so that what it records is
// what the CLI read.
func UploadEvidence(cfg *config.Config, validationRunId string, file string, source EvidenceSource) error {

	// If config is disabled, skips the call
	if cfg.Disabled {
		return nil
	}

	digest, err := fileSha256(file)
	if err != nil {
		return err
	}

	fields := map[string]string{
		"externalDigest": "sha256:" + digest,
	}
	// Yontrack otherwise takes the type of the part, which is sniffed from
	// the content, and cannot tell JSON from text, for example.
	if mediaType := mime.TypeByExtension(filepath.Ext(file)); mediaType != "" {
		fields["mediaType"] = mediaType
	}
	if source.Tool != "" {
		fields["sourceTool"] = source.Tool
	}
	if source.Version != "" {
		fields["sourceVersion"] = source.Version
	}
	if source.URL != "" {
		fields["sourceUrl"] = source.URL
	}

	// The file is given by its path, so that a retry sends it again from the start
	resp, err := newClient(cfg).R().
		SetFile("file", file).
		SetFormData(fields).
		Post(cfg.URL + "/rest/extension/audit-trail/validation-runs/" + validationRunId + "/evidence")
	if err != nil {
		return err
	}
	if resp.IsError() {
		refusal := &EvidenceRefusedError{Status: resp.StatusCode()}
		var body struct {
			Code    string
			Message string
		}
		if json.Unmarshal(resp.Body(), &body) == nil && body.Message != "" {
			refusal.Code = body.Code
			refusal.Message = body.Message
		} else {
			refusal.Message = strings.TrimSpace(string(resp.Body()))
			if refusal.Message == "" {
				refusal.Message = resp.Status()
			}
		}
		return refusal
	}
	return nil
}

func fileSha256(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
