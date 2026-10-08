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

// validatePercentageCmd represents the validatePercentage command
var validatePercentageCmd = &cobra.Command{
	Use:     "percentage",
	Aliases: []string{"percent"},
	Short:   "Validation with percentage data",
	Long: `Validation with percentage data.

For example:

    yontrack validate -p PROJECT -b BRANCH -n BUILD -v VALIDATION percentage --value 87
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

		// Evidence
		evidence, err := getValidateEvidence(cmd)
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
			ValidateBuildWithPercentage struct {
				ValidationRun struct {
					Id string
				}
				Errors []struct {
					Message string
				}
			}
		}

		// Runs the mutation
		if err := client.GraphQLCall(cfg, `
			mutation ValidateBuildWithPercentage(
				$project: String!,
				$branch: String!,
				$build: String!,
				$validationStamp: String!,
				$description: String!,
				$runInfo: RunInfoInput,
				$value: Int!
			) {
				validateBuildWithPercentage(input: {
					project: $project,
					branch: $branch,
					build: $build,
					validation: $validationStamp,
					description: $description,
					runInfo: $runInfo,
					value: $value
				}) {
					validationRun {
						id
					}
					errors {
						message
					}
				}
			}
		`, map[string]interface{}{
			"project":         project,
			"branch":          branch,
			"build":           build,
			"validationStamp": validation,
			"description":     description,
			"runInfo":         runInfo,
			"value":           value,
		}, &payload); err != nil {
			return err
		}

		// Checks for errors
		if err := client.CheckDataErrors(payload.ValidateBuildWithPercentage.Errors); err != nil {
			return err
		}

		// Evidence
		return evidence.attach(cmd, cfg, payload.ValidateBuildWithPercentage.ValidationRun.Id)
	},
}

func init() {
	validateCmd.AddCommand(validatePercentageCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// validatePercentageCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// validatePercentageCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	validatePercentageCmd.Flags().Int("value", 0, "Percentage value")

	// Run info arguments
	InitRunInfoCommandFlags(validatePercentageCmd)
}
