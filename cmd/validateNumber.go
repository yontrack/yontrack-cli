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
	"yontrack/utils"

	"github.com/spf13/cobra"

	client "yontrack/client"
	config "yontrack/config"
)

// numberValidationDataType is the FQCN of the number (threshold) validation data type.
//
// There is no dedicated mutation for this data type, so the generic 'createValidationRun'
// mutation is used, with the data type identified by its fully qualified class name. This
// is accepted by all supported versions of Yontrack.
const numberValidationDataType = "net.nemerosa.ontrack.extension.general.validation.ThresholdNumberValidationDataType"

// validateNumberCmd represents the validateNumber command
var validateNumberCmd = &cobra.Command{
	Use:   "number",
	Short: "Validation with number data",
	Long: `Validation with number data.

For example:

    yontrack validate -p PROJECT -b BRANCH -n BUILD -v VALIDATION number --value 0

The '--value' flag is required: for a validation stamp counting issues, omitting it
would otherwise silently record a value of 0.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		project, branch, build, err := utils.GetProjectBranchBuildFlags(cmd, false, true)
		if err != nil {
			return err
		}

		validation, err := cmd.Flags().GetString("validation")
		if err != nil {
			return err
		}

		description, err := cmd.Flags().GetString("description")
		if err != nil {
			return err
		}

		runInfo, err := GetRunInfo(cmd)
		if err != nil {
			return err
		}

		value, err := cmd.Flags().GetInt("value")
		if err != nil {
			return err
		}

		// Get the configuration
		cfg, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
		}

		// Variables
		variables := make(map[string]interface{})
		variables["project"] = project
		variables["branch"] = branch
		variables["build"] = build
		variables["validationStamp"] = validation
		variables["dataTypeId"] = numberValidationDataType
		// The data type expects its form shape, not a bare number
		variables["data"] = map[string]interface{}{
			"value": value,
		}

		// Description variable
		if description != "" {
			variables["description"] = description
		}

		// Run info
		if runInfo != nil {
			variables["runInfo"] = runInfo
		}

		// Mutation payload
		var payload struct {
			CreateValidationRun struct {
				Errors []struct {
					Message string
				}
			}
		}

		// Runs the mutation
		if err := client.GraphQLCall(cfg, `
			mutation CreateValidationRun(
				$project: String!,
				$branch: String!,
				$build: String!,
				$validationStamp: String!,
				$description: String,
				$runInfo: RunInfoInput,
				$dataTypeId: String,
				$data: JSON
			) {
				createValidationRun(input: {
					project: $project,
					branch: $branch,
					build: $build,
					validationStamp: $validationStamp,
					description: $description,
					dataTypeId: $dataTypeId,
					data: $data,
					runInfo: $runInfo
				}) {
					errors {
						message
					}
				}
			}
		`, variables, &payload); err != nil {
			return err
		}

		// Checks for errors
		if err := client.CheckDataErrors(payload.CreateValidationRun.Errors); err != nil {
			return err
		}

		// OK
		return nil
	},
}

func init() {
	validateCmd.AddCommand(validateNumberCmd)

	validateNumberCmd.Flags().Int("value", 0, "Number value (required)")
	_ = validateNumberCmd.MarkFlagRequired("value")

	// Run info arguments
	InitRunInfoCommandFlags(validateNumberCmd)
}
