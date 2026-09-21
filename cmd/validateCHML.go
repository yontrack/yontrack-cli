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

// validateCHMLCmd represents the validateCHML command
var validateCHMLCmd = &cobra.Command{
	Use:   "chml",
	Short: "Validation with CHML data",
	Long: `Validation with CHML data.

For example:

    yontrack validate -p PROJECT -b BRANCH -n BUILD -v VALIDATION chml --critical 1 --high 2
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		variables, err := validateCHMLVariables(cmd)
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
			ValidateBuildWithCHML struct {
				Errors []struct {
					Message string
				}
			}
		}

		// Runs the mutation
		if err := client.GraphQLCall(cfg, `
			mutation ValidateBuildWithCHML(
				$project: String!,
				$branch: String!,
				$build: String!,
				$validationStamp: String!,
				$description: String!,
				$runInfo: RunInfoInput,
				$critical: Int!,
				$high: Int!,
				$medium: Int!,
				$low: Int!
			) {
				validateBuildWithCHML(input: {
					project: $project,
					branch: $branch,
					build: $build,
					validation: $validationStamp,
					description: $description,
					runInfo: $runInfo,
					critical: $critical,
					high: $high,
					medium: $medium,
					low: $low
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
		if err := client.CheckDataErrors(payload.ValidateBuildWithCHML.Errors); err != nil {
			return err
		}

		// OK
		return nil
	},
}

// validateCHMLVariables reads the flags of 'validate chml' into the variables
// of its mutation. The build falls back to YONTRACK_BUILD_NAME, as for every
// other 'validate' subcommand.
func validateCHMLVariables(cmd *cobra.Command) (map[string]interface{}, error) {
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

	critical, err := cmd.Flags().GetInt("critical")
	if err != nil {
		return nil, err
	}

	high, err := cmd.Flags().GetInt("high")
	if err != nil {
		return nil, err
	}

	medium, err := cmd.Flags().GetInt("medium")
	if err != nil {
		return nil, err
	}

	low, err := cmd.Flags().GetInt("low")
	if err != nil {
		return nil, err
	}

	runInfo, err := GetRunInfo(cmd)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"project":         project,
		"branch":          branch,
		"build":           build,
		"validationStamp": validation,
		"description":     description,
		"runInfo":         runInfo,
		"critical":        critical,
		"high":            high,
		"medium":          medium,
		"low":             low,
	}, nil
}

func init() {
	validateCmd.AddCommand(validateCHMLCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// validateCHMLCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	validateCHMLCmd.Flags().Int("critical", 0, "Number of critical issues")
	validateCHMLCmd.Flags().Int("high", 0, "Number of high issues")
	validateCHMLCmd.Flags().Int("medium", 0, "Number of medium issues")
	validateCHMLCmd.Flags().Int("low", 0, "Number of low issues")

	// Run info arguments
	InitRunInfoCommandFlags(validateCHMLCmd)
}
