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
	client "yontrack/client"
	config "yontrack/config"

	"github.com/spf13/cobra"
)

// gitlabPropertyVariables returns the variables for the
// setProjectGitLabConfigurationProperty mutation. The repository is the full GitLab project
// path and is passed through as-is: GitLab subgroups make it arbitrarily deep, so there is no
// workspace or organization to split out of it.
func gitlabPropertyVariables(project string, configuration string, repository string, indexationInterval int, issueService string) map[string]interface{} {
	return map[string]interface{}{
		"project":                             project,
		"configuration":                       configuration,
		"repository":                          repository,
		"indexationInterval":                  indexationInterval,
		"issueServiceConfigurationIdentifier": issueService,
	}
}

// projectSetPropertyGitLabCmd represents the projectSetPropertyGitLab command
var projectSetPropertyGitLabCmd = &cobra.Command{
	Use:   "gitlab",
	Short: "Configures a project to use a GitLab repository",
	Long: `Configures a project to use a GitLab repository.

The --repository option is the full GitLab project path, like group/project or
group/subgroup/project: subgroups are normal on GitLab and the path can be arbitrarily deep.

Example:

	yontrack project set-property --project PROJECT gitlab --configuration gitlab.com --repository my-group/my-subgroup/my-project --issue-service self
	`,
	RunE: func(cmd *cobra.Command, args []string) error {

		project, err := cmd.Flags().GetString("project")
		if err != nil {
			return err
		}

		configuration, err := cmd.Flags().GetString("configuration")
		if err != nil {
			return err
		}

		repository, err := cmd.Flags().GetString("repository")
		if err != nil {
			return err
		}

		indexation, err := cmd.Flags().GetInt("indexation")
		if err != nil {
			return err
		}

		issueService, err := cmd.Flags().GetString("issue-service")
		if err != nil {
			return err
		}

		cfg, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
		}

		var data struct {
			SetProjectGitLabConfigurationProperty struct {
				Errors []struct {
					Message string
				}
			}
		}
		if err := client.GraphQLCall(cfg, `
			mutation SetProjectGitLabConfigurationProperty(
				$project: String!,
				$configuration: String!,
				$repository: String!,
				$indexationInterval: Int,
				$issueServiceConfigurationIdentifier: String
			) {
				setProjectGitLabConfigurationProperty(input: {
					project: $project,
					configuration: $configuration,
					repository: $repository,
					indexationInterval: $indexationInterval,
					issueServiceConfigurationIdentifier: $issueServiceConfigurationIdentifier
				}) {
					errors {
						message
					}
				}
			}
		`, gitlabPropertyVariables(project, configuration, repository, indexation, issueService), &data); err != nil {
			return err
		}

		if err := client.CheckDataErrors(data.SetProjectGitLabConfigurationProperty.Errors); err != nil {
			return err
		}

		// OK
		return nil
	},
}

func init() {
	projectSetPropertyCmd.AddCommand(projectSetPropertyGitLabCmd)

	projectSetPropertyGitLabCmd.Flags().StringP("configuration", "c", "", "Name of the GitLab configuration to use")
	// Only the first backquoted word is taken by Cobra as the flag placeholder, so the deeper
	// path is spelled out without backquotes.
	projectSetPropertyGitLabCmd.Flags().StringP("repository", "r", "", "Full GitLab project path to use, in the form of `group/project`, or group/subgroup/project when subgroups are used")
	projectSetPropertyGitLabCmd.Flags().Int("indexation", 0, "Repository interval to use (in minutes)")
	projectSetPropertyGitLabCmd.Flags().String("issue-service", "", "Issue identifier to use, for example jira//name where name is the name of the JIRA configuration in Yontrack.")

	projectSetPropertyGitLabCmd.MarkFlagRequired("configuration")
	projectSetPropertyGitLabCmd.MarkFlagRequired("repository")
}
