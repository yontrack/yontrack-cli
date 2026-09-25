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

// validationStampSetupCmd represents the validationStampSetup command
var validationStampSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Creates or updates a validation stamp",
	Long: `Creates or updates a validation stamp.

To create a plain validation stamp (without any data type):

	yontrack vs setup --project PROJECT --branch BRANCH --validation STAMP

You can also associate a data type with it, using the JSON representation of the configuration:

	yontrack vs setup --project PROJECT --branch BRANCH --validation STAMP \
		--data-type "net.nemerosa.ontrack.extension.general.validation.CHMLValidationDataType" \
		--data-config '{warningLevel: {level: "HIGH",value:1}, failedLevel:{level:"CRITICAL",value:1}}'

Dedicated commands for the most used data types are also available, see the
subcommands below.
`,
	// Rejects a mistyped subcommand instead of setting up a plain validation stamp
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return setupValidationStamp(cmd)
	},
}

// setupValidationStamp creates or updates a validation stamp, with the data
// type and configuration given by the --data-type and --data-config flags.
func setupValidationStamp(cmd *cobra.Command) error {
	project, branch, err := utils.GetProjectBranchFlags(cmd, false, true)
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

	dataType, err := cmd.Flags().GetString("data-type")
	if err != nil {
		return err
	}

	dataTypeConfig, err := cmd.Flags().GetString("data-config")
	if err != nil {
		return err
	}

	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	return client.SetupValidationStamp(
		cfg,
		project,
		branch,
		validation,
		description,
		dataType,
		dataTypeConfig,
	)
}

func init() {
	validationStampCmd.AddCommand(validationStampSetupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// validationStampSetupCmd.PersistentFlags().String("foo", "", "A help for foo")
	validationStampSetupCmd.PersistentFlags().StringP("validation", "v", "", "Name of the validation stamp")
	validationStampSetupCmd.PersistentFlags().StringP("description", "d", "", "Description for the validation stamp")

	validationStampSetupCmd.MarkPersistentFlagRequired("validation")

	// Local flags: the dedicated commands per data type do not take them
	addDataTypeFlags(validationStampSetupCmd)
}

// addDataTypeFlags registers the flags read by setupValidationStamp.
func addDataTypeFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("data-type", "t", "", "FQCN of the data type")
	cmd.Flags().StringP("data-config", "c", "", "JSON for the data type configuration")
}
