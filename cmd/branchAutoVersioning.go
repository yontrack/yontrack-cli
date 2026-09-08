package cmd

import (
	"encoding/json"
	"yontrack/utils"

	"github.com/spf13/cobra"
	yamljson "sigs.k8s.io/yaml"

	"os"
	"yontrack/client"
	"yontrack/config"
)

type AutoVersioningConfig struct {
	// List of configurations
	Dependencies []map[string]interface{}
}

var branchAutoVersioningCmd = &cobra.Command{
	Use:   "auto-versioning",
	Short: "Management of branch auto-versioning",
	Long: `Management of branch auto-versioning

	yontrack branch auto-versioning --project PROJECT --branch BRANCH --yaml FILE

This sets up the auto-versioning for a branch from a YAML file. The path defaults to ".ontrack/auto-versioning.yaml".

The format of this file is full described in the Yontrack documentation.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		project, branch, err := utils.GetProjectBranchFlags(cmd, false, true)
		if err != nil {
			return err
		}
		avConfigPath, err := cmd.Flags().GetString("yaml")
		if err != nil {
			return err
		}
		if avConfigPath == "" {
			avConfigPath = ".ontrack/auto-versioning.yaml"
		}

		// Reading the AV config file
		buf, err := os.ReadFile(avConfigPath)
		if err != nil {
			return err
		}

		jsonBytes, err := yamljson.YAMLToJSON(buf)
		if err != nil {
			return err
		}

		var root AutoVersioningConfig
		if err := json.Unmarshal(jsonBytes, &root); err != nil {
			return err
		}

		// Getting the list of dependencies
		dependencies := root.Dependencies

		// Configuration
		config, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
		}

		var data struct {
			SetAutoVersioningConfigByName struct {
				Errors []struct {
					Message string
				}
			}
		}

		// GraphQL call
		if err := client.GraphQLCall(config, `
			mutation SetAutoVersioningConfigByName(
			  $project: String!,
			  $branch: String!,
			  $configurations: [AutoVersioningSourceConfigInput!]!,
			) {
			  setAutoVersioningConfigByName(input: {
				project: $project,
				branch: $branch,
				configurations: $configurations
			  }) {
				errors {
				  message
				}
			  }
			}
		`, map[string]interface{}{
			"project":        project,
			"branch":         branch,
			"configurations": dependencies,
		}, &data); err != nil {
			return err
		}

		// Checks errors
		if err := client.CheckDataErrors(data.SetAutoVersioningConfigByName.Errors); err != nil {
			return err
		}

		// OK
		return nil
	},
}

func init() {
	branchCmd.AddCommand(branchAutoVersioningCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// branchSetupCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	branchAutoVersioningCmd.Flags().StringP("project", "p", "", "Project name")
	branchAutoVersioningCmd.Flags().StringP("branch", "b", "", "Branch name or Git branch name")
	branchAutoVersioningCmd.Flags().StringP("yaml", "y", ".ontrack/auto-versioning.yaml", "Path to the YAML file")

	branchAutoVersioningCmd.MarkFlagRequired("project")
	branchAutoVersioningCmd.MarkFlagRequired("branch")
	branchAutoVersioningCmd.MarkFlagRequired("yaml")
}
