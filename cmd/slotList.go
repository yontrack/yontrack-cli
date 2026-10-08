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
	"sort"
	"text/tabwriter"
	"yontrack/client"
	config "yontrack/config"
	"yontrack/utils"

	"github.com/spf13/cobra"
)

// slotListCmd represents the slot list command
var slotListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists the slots of a project, and what each one runs",
	Long: `Lists the slots of a project in every environment, and what each one runs.

    yontrack slot list --project my-project

It prints one row per slot, in the order of the environments: the
environment, the qualifier of the slot (empty for the default one), the slot
ID, and the display name of the build it last deployed - or '—' when it never
completed a deployment.

'--output json' prints the environments and their slots as Yontrack returns
them.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return slotList(cmd)
	},
}

// slotListEnvironment is an environment and the slots of the project in it.
type slotListEnvironment struct {
	Name  string         `json:"name"`
	Order *int           `json:"order"`
	Slots []slotListSlot `json:"slots"`
}

type slotListSlot struct {
	Id                   string `json:"id"`
	Qualifier            string `json:"qualifier"`
	LastDeployedPipeline *struct {
		Build *struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		} `json:"build"`
	} `json:"lastDeployedPipeline"`
}

// deployedBuild is the display name of the build the slot last deployed, or
// a dash when it never completed a deployment.
func (s slotListSlot) deployedBuild() string {
	if s.LastDeployedPipeline == nil || s.LastDeployedPipeline.Build == nil {
		return "—"
	}
	if s.LastDeployedPipeline.Build.DisplayName != "" {
		return s.LastDeployedPipeline.Build.DisplayName
	}
	return s.LastDeployedPipeline.Build.Name
}

func slotList(cmd *cobra.Command) error {
	project, err := utils.GetProjectFlag(cmd)
	if err != nil {
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	if output != "" && output != "json" {
		return fmt.Errorf("unknown output format: %s", output)
	}

	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	// There is no Project.slots: the slots of a project are found through
	// the environments.
	var data struct {
		Environments []slotListEnvironment
	}
	if err := client.GraphQLCall(cfg, `
		query SlotList($project: String!) {
			environments(filter: {projects: [$project]}) {
				name
				order
				slots(projects: [$project]) {
					id
					qualifier
					lastDeployedPipeline {
						build {
							name
							displayName
						}
					}
				}
			}
		}
	`, map[string]interface{}{
		"project": project,
	}, &data); err != nil {
		return err
	}

	environments := data.Environments
	sort.SliceStable(environments, func(i, j int) bool {
		return environmentOrder(environments[i]) < environmentOrder(environments[j])
	})

	out := cmd.OutOrStdout()
	if output == "json" {
		bytes, err := json.MarshalIndent(environments, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "%s\n", bytes)
		return nil
	}

	table := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, environment := range environments {
		for _, slot := range environment.Slots {
			_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", environment.Name, slot.Qualifier, slot.Id, slot.deployedBuild())
		}
	}
	return table.Flush()
}

// environmentOrder is the order of an environment, those without one last.
func environmentOrder(environment slotListEnvironment) int {
	if environment.Order == nil {
		return int(^uint(0) >> 1)
	}
	return *environment.Order
}

func init() {
	slotCmd.AddCommand(slotListCmd)

	slotListCmd.Flags().StringP("project", "p", "", "Name of the project (defaults to YONTRACK_PROJECT_NAME)")
	slotListCmd.Flags().StringP("output", "o", "", "How to output the slots (json). Text by default.")
}
