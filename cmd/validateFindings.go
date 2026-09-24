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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"yontrack/utils"

	"github.com/spf13/cobra"

	client "yontrack/client"
	config "yontrack/config"
)

// validateFindingsCmd represents the validateFindings command
var validateFindingsCmd = &cobra.Command{
	Use:   "findings",
	Short: "Validation with a security findings report",
	Long: `Validation with a security findings report.

The report is sent as is: Yontrack parses it and computes the status of the
validation run. Requires Yontrack 6.0.

For example:

    yontrack validate -p PROJECT -b BRANCH -n BUILD -v VALIDATION findings \
        --format trivy --kind IMAGE --report trivy.json
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		input, err := validateFindingsInput(cmd)
		if err != nil {
			return err
		}

		// Get the configuration
		cfg, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
		}

		// Mutation payload
		var payload struct {
			ValidateBuildWithFindings struct {
				Errors []struct {
					Message string
				}
			}
		}

		// Runs the mutation. The input is passed as a whole, so that Yontrack
		// alone decides how its fields are typed.
		if err := client.GraphQLCall(cfg, `
			mutation ValidateBuildWithFindings($input: ValidateBuildWithFindingsInput!) {
				validateBuildWithFindings(input: $input) {
					errors {
						message
					}
				}
			}
		`, map[string]interface{}{"input": input}, &payload); err != nil {
			return err
		}

		// Checks for errors
		return client.CheckDataErrors(payload.ValidateBuildWithFindings.Errors)
	},
}

// Formats and kinds of scan accepted by the validateBuildWithFindings mutation
var (
	findingsFormats = []string{"findings", "sarif", "trivy"}
	findingsKinds   = []string{"IMAGE", "CODE", "SECRETS", "DAST", "DEPENDENCIES", "OTHER"}
)

// validateFindingsInput reads the flags of 'validate findings' into the input
// of its mutation. The report is read but not parsed: all parsing is done by
// Yontrack.
func validateFindingsInput(cmd *cobra.Command) (map[string]interface{}, error) {
	project, branch, build, err := utils.GetProjectBranchBuildFlags(cmd, false, true)
	if err != nil {
		return nil, err
	}

	validation, err := cmd.Flags().GetString("validation")
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return nil, err
	}

	// The format is left to Yontrack to check, so that a format it adds does
	// not wait for a release of the CLI.
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, err
	}
	if format == "" {
		return nil, fmt.Errorf("format is required (use --format flag): %s", strings.Join(findingsFormats, ", "))
	}

	kind, err := requiredFlagOneOf(cmd, "kind", findingsKinds)
	if err != nil {
		return nil, err
	}

	scanner, err := cmd.Flags().GetString("scanner")
	if err != nil {
		return nil, err
	}

	reportFile, err := cmd.Flags().GetString("report")
	if err != nil {
		return nil, err
	}
	if reportFile == "" {
		return nil, errors.New("report is required (use --report flag)")
	}
	report, err := os.ReadFile(reportFile)
	if err != nil {
		return nil, fmt.Errorf("cannot read the report: %w", err)
	}
	if !json.Valid(report) {
		return nil, fmt.Errorf("the report %s is not valid JSON", reportFile)
	}

	runInfo, err := GetRunInfo(cmd)
	if err != nil {
		return nil, err
	}

	input := map[string]interface{}{
		"project":     project,
		"branch":      branch,
		"build":       build,
		"validation":  validation,
		"description": description,
		"runInfo":     runInfo,
		"format":      format,
		"kind":        kind,
		"report":      json.RawMessage(report),
	}
	if scanner != "" {
		input["scanner"] = scanner
	}
	return input, nil
}

// requiredFlagOneOf reads a required flag which accepts only the given values.
func requiredFlagOneOf(cmd *cobra.Command, name string, accepted []string) (string, error) {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", err
	}
	list := strings.Join(accepted, ", ")
	if value == "" {
		return "", fmt.Errorf("%s is required (use --%s flag): %s", name, name, list)
	}
	if !slices.Contains(accepted, value) {
		return "", fmt.Errorf("unknown %s %s: %s", name, value, list)
	}
	return value, nil
}

func init() {
	validateCmd.AddCommand(validateFindingsCmd)

	validateFindingsCmd.Flags().String("format", "", "Format of the report: "+strings.Join(findingsFormats, ", ")+" (required)")
	validateFindingsCmd.Flags().String("kind", "", "Kind of scan: "+strings.Join(findingsKinds, ", ")+" (required)")
	validateFindingsCmd.Flags().String("scanner", "", "Name of the scanner, overriding the one read from the report")
	validateFindingsCmd.Flags().String("report", "", "Path to the JSON report file (required)")

	// Run info arguments
	InitRunInfoCommandFlags(validateFindingsCmd)
}
