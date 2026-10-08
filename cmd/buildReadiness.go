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
	"errors"
	"fmt"
	"yontrack/client"
	config "yontrack/config"
	"yontrack/utils"

	"github.com/spf13/cobra"
)

// buildReadinessCmd represents the build readiness command
var buildReadinessCmd = &cobra.Command{
	Use:   "readiness",
	Short: "Tells what a build still lacks to reach a promotion level or a slot",
	Long: `Tells what a build still lacks to reach a promotion level of its branch, or to be
deployed in a slot of its project. Requires Yontrack 6.0.

    yontrack build readiness --project my-project --branch main --build 42 --promotion GOLD
    yontrack build readiness --project my-project --branch main --build 42 --slot <slot-id>

Exactly one of --promotion and --slot is required. Reading the readiness
promotes or deploys nothing.

It prints 'ready', or 'not ready' followed by one line per missing condition:
its kind (VALIDATION, PROMOTION, CHECK, ADMISSION_RULE, MANUAL, AGENT_POLICY),
its name and why it is missing. '--output json' prints the readiness as
Yontrack returns it.

The exit code tells a pipeline what to do:

    0  ready
    2  not ready - wait, or act on what is missing
    1  any error - fix the pipeline
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return buildReadiness(cmd)
	},
}

// readiness is what a build still lacks to reach a promotion level or a slot.
type readiness struct {
	Ready   bool               `json:"ready"`
	Missing []readinessMissing `json:"missing"`
}

// readinessMissing is one condition a build does not meet yet.
type readinessMissing struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

func buildReadiness(cmd *cobra.Command) error {
	project, branch, build, err := utils.GetProjectBranchBuildFlags(cmd, false, true)
	if err != nil {
		return err
	}
	promotion, err := cmd.Flags().GetString("promotion")
	if err != nil {
		return err
	}
	slotId, err := cmd.Flags().GetString("slot")
	if err != nil {
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	if err := checkReadinessTarget(promotion, slotId); err != nil {
		return err
	}
	if output != "" && output != "json" {
		return fmt.Errorf("unknown output format: %s", output)
	}

	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	variables := map[string]interface{}{
		"project": project,
		"branch":  branch,
		"build":   build,
	}
	if promotion != "" {
		variables["promotionLevel"] = promotion
	} else {
		variables["slotId"] = slotId
	}

	var data struct {
		Builds []struct {
			Readiness readiness
		}
	}
	if err := client.GraphQLCall(cfg, `
		query BuildReadiness($project: String!, $branch: String!, $build: String!, $promotionLevel: String, $slotId: String) {
			builds(project: $project, branch: $branch, name: $build) {
				readiness(promotionLevel: $promotionLevel, slotId: $slotId) {
					ready
					missing {
						kind
						name
						message
					}
				}
			}
		}
	`, variables, &data); err != nil {
		return err
	}
	if len(data.Builds) == 0 {
		return fmt.Errorf("no build %s in branch %s of project %s", build, branch, project)
	}
	result := data.Builds[0].Readiness

	out := cmd.OutOrStdout()
	if output == "json" {
		bytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "%s\n", bytes)
	} else if result.Ready {
		_, _ = fmt.Fprintln(out, "ready")
	} else {
		_, _ = fmt.Fprintln(out, "not ready")
		for _, missing := range result.Missing {
			_, _ = fmt.Fprintf(out, "  - %s %s: %s\n", missing.Kind, missing.Name, missing.Message)
		}
	}

	if !result.Ready {
		return exitWith(cmd, 2, "not ready")
	}
	return nil
}

// checkReadinessTarget makes sure exactly one target was given.
func checkReadinessTarget(promotion, slotId string) error {
	if promotion == "" && slotId == "" {
		return errors.New("one of --promotion or --slot is required")
	}
	if promotion != "" && slotId != "" {
		return errors.New("--promotion and --slot are mutually exclusive")
	}
	return nil
}

func init() {
	buildCmd.AddCommand(buildReadinessCmd)

	buildReadinessCmd.Flags().StringP("project", "p", "", "Name of the project (defaults to YONTRACK_PROJECT_NAME)")
	buildReadinessCmd.Flags().StringP("branch", "b", "", "Name of the branch (defaults to YONTRACK_BRANCH_NAME)")
	buildReadinessCmd.Flags().StringP("build", "n", "", "Name of the build (defaults to YONTRACK_BUILD_NAME)")
	buildReadinessCmd.Flags().String("promotion", "", "Promotion level of the branch to reach. Exclusive with --slot.")
	buildReadinessCmd.Flags().String("slot", "", "ID of the slot of the project to be deployed in. Exclusive with --promotion.")
	buildReadinessCmd.Flags().StringP("output", "o", "", "How to output the readiness (json). Text by default.")
}
