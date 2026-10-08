/*
Copyright © 2021 Damien Coraboeuf <damien.coraboeuf@nemerosa.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"yontrack/client"
	config "yontrack/config"

	"github.com/spf13/cobra"
)

// validateEvidence is what the 'validate' commands attach as evidence to the
// validation run they create.
type validateEvidence struct {
	// Files to attach - those which could be read
	files []string
	// Whether a missing evidence is only a warning
	optional bool
	// Where the evidence comes from
	source client.EvidenceSource
}

// initValidateEvidenceFlags declares the evidence flags on the 'validate'
// command, for it and all its subcommands.
func initValidateEvidenceFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringArray("evidence", []string{}, "Path to a file to attach as evidence to the validation run (repeatable, requires Yontrack 6.0)")
	cmd.PersistentFlags().Bool("evidence-optional", false, "Only warn, instead of failing, when an evidence cannot be read or is refused by Yontrack")
	cmd.PersistentFlags().String("evidence-tool", "", "Tool which produced the evidence, like trivy")
	cmd.PersistentFlags().String("evidence-tool-version", "", "Version of the tool which produced the evidence")
	cmd.PersistentFlags().String("evidence-source-url", "", "Where the evidence was produced, like the URL of the CI job (HTTP or HTTPS)")
}

// getValidateEvidence reads the evidence flags. It is called before the
// validation run is created, so that an evidence which cannot be read fails
// the command before anything is recorded - or, with --evidence-optional, is
// left out with a warning.
func getValidateEvidence(cmd *cobra.Command) (*validateEvidence, error) {
	files, err := cmd.Flags().GetStringArray("evidence")
	if err != nil {
		return nil, err
	}
	optional, err := cmd.Flags().GetBool("evidence-optional")
	if err != nil {
		return nil, err
	}
	tool, err := cmd.Flags().GetString("evidence-tool")
	if err != nil {
		return nil, err
	}
	version, err := cmd.Flags().GetString("evidence-tool-version")
	if err != nil {
		return nil, err
	}
	url, err := cmd.Flags().GetString("evidence-source-url")
	if err != nil {
		return nil, err
	}

	evidence := &validateEvidence{
		optional: optional,
		source:   client.EvidenceSource{Tool: tool, Version: version, URL: url},
	}
	for _, file := range files {
		if err := checkEvidenceFile(file); err != nil {
			if !optional {
				return nil, err
			}
			warnEvidence(cmd, err)
			continue
		}
		evidence.files = append(evidence.files, file)
	}
	return evidence, nil
}

// checkEvidenceFile checks that an evidence can be sent: a file, which is not
// empty.
func checkEvidenceFile(file string) error {
	info, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("cannot read the evidence %s: %w", file, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("cannot read the evidence %s: not a file", file)
	}
	if info.Size() == 0 {
		return fmt.Errorf("cannot read the evidence %s: the file is empty", file)
	}
	return nil
}

// attach uploads every evidence to the validation run which was just
// created. A missing evidence is an audit gap: every refusal fails the
// command, unless the evidence is optional, in which case it is a warning.
func (e *validateEvidence) attach(cmd *cobra.Command, cfg *config.Config, validationRunId string) error {
	if len(e.files) == 0 || cfg.Disabled {
		return nil
	}
	var failures []string
	for _, file := range e.files {
		var err error
		if validationRunId == "" {
			err = fmt.Errorf("evidence %s was not attached: Yontrack did not return the validation run", file)
		} else if uploadErr := client.UploadEvidence(cfg, validationRunId, file, e.source); uploadErr != nil {
			err = fmt.Errorf("evidence %s was not attached to validation run %s: %w", file, validationRunId, uploadErr)
		}
		if err == nil {
			continue
		}
		if e.optional {
			warnEvidence(cmd, err)
		} else {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "\n"))
	}
	return nil
}

func warnEvidence(cmd *cobra.Command, err error) {
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", err)
}
