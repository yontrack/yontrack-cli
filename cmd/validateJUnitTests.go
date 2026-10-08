package cmd

import (
	"yontrack/utils"

	"github.com/spf13/cobra"

	client "yontrack/client"
	"yontrack/cmd/junit"
	config "yontrack/config"
)

var validateJUnitTestsCmd = &cobra.Command{
	Use:   "junit",
	Short: "Validation with JUnit test data",
	Long: `Validation with JUnit XML test data.

For example:

    yontrack validate -p PROJECT -b BRANCH -n BUILD -v VALIDATION junit --pattern "**/results/*.xml" --fail-when-no-results
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

		pattern, err := cmd.Flags().GetString("pattern")
		if err != nil {
			return err
		}

		// Get the configuration
		cfg, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
		}

		// Parsing of JUnit test reports
		passed, skipped, failed, err := junit.GetSummaryJUnitTestReports(pattern)
		if err != nil {
			return err
		}

		failWhenNoResults, err := cmd.Flags().GetBool("fail-when-no-results")
		if err != nil {
			return err
		}

		var status *string
		if failWhenNoResults && passed == 0 && skipped == 0 && failed == 0 {
			FAILED := "FAILED"
			status = &FAILED
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
			status,
		)
		if err != nil {
			return err
		}

		// Evidence
		return evidence.attach(cmd, cfg, validationRunId)
	},
}

func init() {
	validateCmd.AddCommand(validateJUnitTestsCmd)
	validateJUnitTestsCmd.Flags().String("pattern", "", "Pattern (glob) to the JUnit XML tests")
	validateJUnitTestsCmd.Flags().Bool("fail-when-no-results", false, "Fail validation check in case no test results found")
	// Run info arguments
	InitRunInfoCommandFlags(validateJUnitTestsCmd)
}
