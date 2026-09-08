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

// branchSetupCmd represents the branchSetup command
var branchSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Creates a branch (and its project) if they don't exist yet.",
	Long: `Creates a branch (and its project) if they don't exist yet.

    yontrack branch setup --project PROJECT --branch BRANCH

The BRANCH name will be adapted to fit Yontrack naming conventions, so you
can directly give the name of the Git branch.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		project, branch, err := utils.GetProjectBranchFlags(cmd, false, true)
		if err != nil {
			return err
		}

		// Project auto validation stamps
		autoCreateVS, err := cmd.Flags().GetBool("auto-create-vs")
		if err != nil {
			return err
		}
		autoCreateVSAlways, err := cmd.Flags().GetBool("auto-create-vs-always")
		if err != nil {
			return err
		}
		if autoCreateVSAlways {
			autoCreateVS = true
		}

		// Project auto promotion levels
		autoCreatePL, err := cmd.Flags().GetBool("auto-create-pl")
		if err != nil {
			return err
		}

		// Configuration
		config, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
		}
		// Creates or get the project
		var data struct {
			CreateProjectOrGet struct {
				Project struct {
					ID int
				}
				Errors []struct {
					Message string
				}
			}
			CreateBranchOrGet struct {
				Branch struct {
					ID int
				}
				Errors []struct {
					Message string
				}
			}
			SetProjectAutoValidationStampProperty struct {
				Errors []struct {
					Message string
				}
			}
			SetProjectAutoPromotionLevelProperty struct {
				Errors []struct {
					Message string
				}
			}
		}
		if err := client.GraphQLCall(config, `
			mutation ProjectSetup(
				$project: String!, 
				$branch: String!,
				$autoCreateVS: Boolean!,
				$autoCreateVSIfNotPredefined: Boolean!,
				$autoCreatePL: Boolean!
			) {
				createProjectOrGet(input: {name: $project}) {
					errors {
					message
					}
				}
				createBranchOrGet(input: {projectName: $project, name: $branch}) {
					errors {
					message
					}
				}
				setProjectAutoValidationStampProperty(input: {
					project: $project,
					isAutoCreate: $autoCreateVS,
					isAutoCreateIfNotPredefined: $autoCreateVSIfNotPredefined
				}) {
					errors {
						message
					}
				}
				setProjectAutoPromotionLevelProperty(input: {
					project: $project,
					isAutoCreate: $autoCreatePL
				}) {
					errors {
						message
					}
				}
			}
		`, map[string]interface{}{
			"project":                     project,
			"branch":                      branch,
			"autoCreateVS":                autoCreateVS,
			"autoCreateVSIfNotPredefined": autoCreateVSAlways,
			"autoCreatePL":                autoCreatePL,
		}, &data); err != nil {
			return err
		}

		// Checks errors for the project
		if err := client.CheckDataErrors(data.CreateProjectOrGet.Errors); err != nil {
			return err
		}
		// Checks errors for the branch
		if err := client.CheckDataErrors(data.CreateBranchOrGet.Errors); err != nil {
			return err
		}
		// Checks errors for the project auto validation stamp propetyu
		if err := client.CheckDataErrors(data.SetProjectAutoValidationStampProperty.Errors); err != nil {
			return err
		}
		// Checks errors for the project auto promotion level propetyu
		if err := client.CheckDataErrors(data.SetProjectAutoPromotionLevelProperty.Errors); err != nil {
			return err
		}

		// OK
		return nil
	},
}

func init() {
	branchCmd.AddCommand(branchSetupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// branchSetupCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	branchSetupCmd.Flags().StringP("project", "p", "", "Project name")
	branchSetupCmd.Flags().StringP("branch", "b", "", "Branch name or Git branch name")

	branchSetupCmd.Flags().Bool("auto-create-vs", true, "Auto creation of validation stamps if they are predefined")
	branchSetupCmd.Flags().Bool("auto-create-vs-always", false, "Auto creation of validation stamps even if they are not predefined")
	branchSetupCmd.Flags().Bool("auto-create-pl", true, "Auto creation of promotion levels if they are predefined")

	branchSetupCmd.MarkFlagRequired("project")
	branchSetupCmd.MarkFlagRequired("branch")
}
