package cmd

import (
	"encoding/json"
	"errors"
	"yontrack/utils"

	"github.com/spf13/cobra"

	client "yontrack/client"
	config "yontrack/config"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validates a build for a validation stamp",
	Long: `Validates a build for a validation stamp.

The simplest form is:

    yontrack validate -p PROJECT -b BRANCH -n BUILD -v VALIDATION -s STATUS

where 'STATUS' is a valid Yontrack status, like 'PASSED', 'WARNING' or 'FAILED'.

In case there is some data to be passed to the validation:

	yontrack validate -p PROJECT -b BRANCH -n BUILD -v VALIDATION \
		--data-type net.nemerosa.ontrack.extension.general.validation.TestSummaryValidationDataType \
		--data {passed: 1, skipped: 2, failed: 3}

In this case, there is no need to pass the status but it could still be forced using the '-s STATUS' flag.

Note that subcommands, dedicated to the most common types are also available. For example:

    yontrack validate -p PROJECT -b BRANCH -n BUILD -v VALIDATION tests --passed 1 --skipped 2 --failed 3

Type 'yontrack validate --help' to get a list of all options.
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
		if validation == "" {
			return errors.New("validation is required (use --validation flag)")
		}

		description, err := cmd.Flags().GetString("description")
		if err != nil {
			return err
		}

		status, err := cmd.Flags().GetString("status")
		if err != nil {
			return err
		}

		dataType, err := cmd.Flags().GetString("data-type")
		if err != nil {
			return err
		}

		data, err := cmd.Flags().GetString("data")
		if err != nil {
			return err
		}

		// Status is required if no data
		if status == "" {
			if dataType == "" {
				return errors.New("Status is required if no data is provided.")
			}
		}

		// Data type is required if some data is provided
		if data != "" {
			if dataType == "" {
				return errors.New("Data type is required if some data is provided")
			}
		}

		// Run info
		runInfo, err := GetRunInfo(cmd)
		if err != nil {
			return err
		}

		// Query
		query := `
			mutation CreateValidationRun(
				$project: String!,
				$branch: String!,
				$build: String!,
				$validationStamp: String!,
				$validationRunStatus: String,
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
					validationRunStatus: $validationRunStatus,
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
		`

		// Variables
		variables := make(map[string]interface{})
		variables["project"] = project
		variables["branch"] = branch
		variables["build"] = build
		variables["validationStamp"] = validation

		// Status variable
		if status != "" {
			variables["validationRunStatus"] = status
		}

		// Description variable
		if description != "" {
			variables["description"] = description
		}

		// Data type ID variable
		if dataType != "" {
			variables["dataTypeId"] = dataType
		}

		// Data variable
		if data != "" {
			var dataJson interface{}
			if err := json.Unmarshal([]byte(data), &dataJson); err != nil {
				return err
			}
			variables["data"] = dataJson
		}

		// Run info
		if runInfo != nil {
			variables["runInfo"] = runInfo
		}

		// Get the configuration
		cfg, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
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
		if err := client.GraphQLCall(cfg, query, variables, &payload); err != nil {
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
	rootCmd.AddCommand(validateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	validateCmd.PersistentFlags().StringP("project", "p", "", "Name of the project")
	validateCmd.PersistentFlags().StringP("branch", "b", "", "Name of the branch")
	validateCmd.PersistentFlags().StringP("build", "n", "", "Name of the build")
	validateCmd.PersistentFlags().StringP("validation", "v", "", "Name of the validation stamp")
	validateCmd.PersistentFlags().StringP("description", "d", "", "Description for the validation")

	_ = validateCmd.MarkPersistentFlagRequired("validation")

	// Run info arguments
	InitRunInfoCommandFlags(validateCmd)

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	validateCmd.Flags().StringP("status", "s", "", "ID of the status (required if no data)")
	validateCmd.Flags().StringP("data-type", "t", "", "FQCN of the validation data type")
	validateCmd.Flags().StringP("data", "o", "", "JSON representation of the validation data")
}
