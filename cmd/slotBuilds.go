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
	"encoding/json"
	"fmt"
	"yontrack/client"
	config "yontrack/config"

	"github.com/spf13/cobra"
)

// slotBuildsCmd represents the slot builds command
var slotBuildsCmd = &cobra.Command{
	Use:   "builds",
	Short: "Lists the builds which can be deployed into a slot",
	Long: `Lists the builds which can be deployed into the slot of a project in an
environment, most recent first:

    yontrack slot builds --project my-project --environment production

By default, only the deployable builds are listed: the ones meeting all the
admission rules of the slot. A build can be eligible for a slot without being
deployable yet - not promoted yet, for example. To list every eligible build,
including those:

    yontrack slot builds --project my-project --environment production --all

By default, the build names are printed, one per line. To get their IDs and
display names too, use the JSON output:

    yontrack slot builds --project my-project --environment production --output json
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return slotBuilds(cmd)
	},
}

func slotBuilds(cmd *cobra.Command) error {
	project, err := cmd.Flags().GetString("project")
	if err != nil {
		return err
	}
	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		return err
	}
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}
	count, err := cmd.Flags().GetInt("count")
	if err != nil {
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	builds, err := getSlotEligibleBuilds(cfg, project, environment, !all, count)
	if err != nil {
		return err
	}

	switch output {
	case "name":
		for _, build := range builds {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), build.Name)
		}
	case "json":
		bytes, err := json.MarshalIndent(builds, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", bytes)
	default:
		return fmt.Errorf("unknown output format: %s", output)
	}

	return nil
}

// getSlotEligibleBuilds lists the builds eligible for the slot of a project in
// an environment, restricted to the deployable ones if asked. 'deployable' is
// always passed, since its default on the server changed.
func getSlotEligibleBuilds(cfg *config.Config, project string, environment string, deployable bool, size int) ([]slotBuild, error) {
	var data struct {
		EnvironmentByName *struct {
			Slots []struct {
				EligibleBuilds *struct {
					PageItems []slotBuild
				}
			}
		}
	}

	if err := client.GraphQLCall(cfg, `
		query GetSlotEligibleBuilds($environment: String!, $project: String!, $deployable: Boolean!, $size: Int!) {
			environmentByName(name: $environment) {
				slots(projects: [$project]) {
					eligibleBuilds(deployable: $deployable, size: $size) {
						pageItems {
							id
							name
							displayName
						}
					}
				}
			}
		}
	`, map[string]interface{}{
		"environment": environment,
		"project":     project,
		"deployable":  deployable,
		"size":        size,
	}, &data); err != nil {
		return nil, err
	}

	if data.EnvironmentByName == nil {
		return nil, fmt.Errorf("no environment named %s", environment)
	}
	if len(data.EnvironmentByName.Slots) == 0 {
		return nil, fmt.Errorf("no slot for project %s in environment %s", project, environment)
	}

	builds := []slotBuild{}
	if eligible := data.EnvironmentByName.Slots[0].EligibleBuilds; eligible != nil {
		builds = append(builds, eligible.PageItems...)
	}
	return builds, nil
}

func init() {
	slotCmd.AddCommand(slotBuildsCmd)

	slotBuildsCmd.Flags().StringP("project", "p", "", "Name of the project")
	slotBuildsCmd.Flags().StringP("environment", "e", "", "Name of the environment")
	slotBuildsCmd.Flags().Bool("all", false, "Lists every eligible build, including the ones not deployable yet")
	slotBuildsCmd.Flags().IntP("count", "n", 10, "Maximum number of builds to list")
	slotBuildsCmd.Flags().StringP("output", "o", "name", "How to display the builds (name, json)")

	_ = slotBuildsCmd.MarkFlagRequired("project")
	_ = slotBuildsCmd.MarkFlagRequired("environment")
}
