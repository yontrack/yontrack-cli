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

// projectSetPropertyBitbucketCloudCmd represents the projectSetPropertyBitbucketCloud command
var projectSetPropertyBitbucketCloudCmd = &cobra.Command{
	Use:   "bitbucket-cloud",
	Short: "Configures a project to use a Bitbucket Cloud repository",
	Long: `Configures a project to use a Bitbucket Cloud repository.

Example:

	yontrack project set-property --project PROJECT bitbucket-cloud --configuration Bitbucket-Cloud --repository my-repository --issue-service jira//my-jira`,

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
			SetProjectBitbucketCloudConfigurationProperty struct {
				Errors []struct {
					Message string
				}
			}
		}
		if err := client.GraphQLCall(cfg, `
			mutation SetProjectBitbucketCloudConfigurationProperty(
				$project: String!,
				$configuration: String!,
				$repository: String!,
				$indexationInterval: Int,
				$issueServiceConfigurationIdentifier: String
			) {
				setProjectBitbucketCloudConfigurationProperty(input: {
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
		`, map[string]interface{}{
			"project":                             project,
			"configuration":                       configuration,
			"repository":                          repository,
			"indexationInterval":                  indexation,
			"issueServiceConfigurationIdentifier": issueService,
		}, &data); err != nil {
			return err
		}

		if err := client.CheckDataErrors(data.SetProjectBitbucketCloudConfigurationProperty.Errors); err != nil {
			return err
		}

		// OK
		return nil
	},
}

func init() {
	projectSetPropertyCmd.AddCommand(projectSetPropertyBitbucketCloudCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// projectSetPropertyBitbucketCloudCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// projectSetPropertyBitbucketCloudCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	projectSetPropertyBitbucketCloudCmd.Flags().StringP("configuration", "c", "", "Name of the Bitbucket Cloud configuration to use")
	projectSetPropertyBitbucketCloudCmd.Flags().StringP("repository", "r", "", "Bitbucket Cloud repository to use")
	projectSetPropertyBitbucketCloudCmd.Flags().Int("indexation", 0, "Repository interval to use (in minutes)")
	projectSetPropertyBitbucketCloudCmd.Flags().String("issue-service", "", "Issue identifier to use, for example jira//name where name is the name of the JIRA configuration in Yontrack.")

	projectSetPropertyBitbucketCloudCmd.MarkFlagRequired("configuration")
	projectSetPropertyBitbucketCloudCmd.MarkFlagRequired("repository")
}
