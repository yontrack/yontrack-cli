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

// validateTestsCmd represents the validateTests command
var validateTestsCmd = &cobra.Command{
	Use:   "tests",
	Short: "Validation with test data",
	Long: `Validation with test data.

For example:

    yontrack validate -p PROJECT -b BRANCH -n BUILD -v VALIDATION tests --passed 1 --skipped 2 --failed 3
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

		evidence, err := getValidateEvidence(cmd)
		if err != nil {
			return err
		}

		passed, err := cmd.Flags().GetInt("passed")
		if err != nil {
			return err
		}

		skipped, err := cmd.Flags().GetInt("skipped")
		if err != nil {
			return err
		}

		failed, err := cmd.Flags().GetInt("failed")
		if err != nil {
			return err
		}

		// Get the configuration
		cfg, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
		}

		// Call
		validationRunId, err := client.ValidateWithTests(
			cfg,
			project,
			branch,
			build,
			validation,
			description,
			runInfo,
			passed,
			skipped,
			failed,
			nil,
		)
		if err != nil {
			return err
		}

		// Evidence
		return evidence.attach(cmd, cfg, validationRunId)
	},
}

func init() {
	validateCmd.AddCommand(validateTestsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// validateTestsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// validateTestsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	validateTestsCmd.Flags().Int("passed", 0, "Number of passed tests")
	validateTestsCmd.Flags().Int("skipped", 0, "Number of skipped tests")
	validateTestsCmd.Flags().Int("failed", 0, "Number of failed tests")

	// Run info arguments
	InitRunInfoCommandFlags(validateTestsCmd)
}
