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
	"strconv"
	"strings"
	"yontrack/client"
	config "yontrack/config"
	"yontrack/utils"

	"github.com/spf13/cobra"
)

// buildChangelogSinceDeployedCmd represents the build changelog since-deployed command
var buildChangelogSinceDeployedCmd = &cobra.Command{
	Use:   "since-deployed",
	Short: "Lists the commits between what a slot runs and a candidate build",
	Long: `Lists the commits between the build a slot last deployed and a candidate build:
what deploying the candidate would change. Requires Yontrack 6.0.

    yontrack build changelog since-deployed --project my-project --branch main --build 45 --slot <slot-id>

The deployed build is the one of the last deployment of the slot which
completed. The command fails when nothing was deployed in the slot yet.

It prints one line per commit: its short ID, its subject, its author, and the
assistants (agents) which helped write it, with the markers which named them.
'--output json' prints the change log as Yontrack returns it.

To get the ID of a slot:

    yontrack slot get --project my-project --environment production
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return buildChangelogSinceDeployed(cmd)
	},
}

// sinceDeployedChangeLog is the change log between two builds, with the
// assistants of each commit.
type sinceDeployedChangeLog struct {
	Commits []struct {
		Commit sinceDeployedCommit `json:"commit"`
	} `json:"commits"`
}

type sinceDeployedCommit struct {
	Id         string                   `json:"id"`
	ShortId    string                   `json:"shortId"`
	Message    string                   `json:"message"`
	Author     string                   `json:"author"`
	Assistants []sinceDeployedAssistant `json:"assistants"`
}

type sinceDeployedAssistant struct {
	Name        string   `json:"name"`
	Markers     []string `json:"markers"`
	SessionLink *string  `json:"sessionLink"`
}

func buildChangelogSinceDeployed(cmd *cobra.Command) error {
	project, branch, build, err := utils.GetProjectBranchBuildFlags(cmd, false, true)
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
	if slotId == "" {
		return errors.New("--slot is required")
	}
	if output != "" && output != "json" {
		return fmt.Errorf("unknown output format: %s", output)
	}

	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	from, to, err := sinceDeployedBoundaries(cfg, project, branch, build, slotId)
	if err != nil {
		return err
	}

	var data struct {
		ScmChangeLog *sinceDeployedChangeLog
	}
	if err := client.GraphQLCall(cfg, `
		query SinceDeployedChangeLog($from: Int!, $to: Int!) {
			scmChangeLog(from: $from, to: $to) {
				commits {
					commit {
						id
						shortId
						message
						author
						assistants {
							name
							markers
							sessionLink
						}
					}
				}
			}
		}
	`, map[string]interface{}{
		"from": from,
		"to":   to,
	}, &data); err != nil {
		return err
	}
	if data.ScmChangeLog == nil {
		return fmt.Errorf("no change log between the build deployed in slot %s and build %s", slotId, build)
	}

	out := cmd.OutOrStdout()
	if output == "json" {
		bytes, err := json.MarshalIndent(data.ScmChangeLog, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "%s\n", bytes)
		return nil
	}
	for _, item := range data.ScmChangeLog.Commits {
		_, _ = fmt.Fprintln(out, sinceDeployedLine(item.Commit))
	}
	return nil
}

// sinceDeployedBoundaries are the IDs of the build the slot last deployed, and
// of the candidate build.
func sinceDeployedBoundaries(cfg *config.Config, project, branch, build, slotId string) (from int, to int, err error) {
	var data struct {
		Builds []struct {
			Id string
		}
		SlotById *struct {
			LastDeployedPipeline *struct {
				Build *struct {
					Id string
				}
			}
		}
	}
	if err := client.GraphQLCall(cfg, `
		query SinceDeployedBoundaries($project: String!, $branch: String!, $build: String!, $slot: String!) {
			builds(project: $project, branch: $branch, name: $build) {
				id
			}
			slotById(id: $slot) {
				lastDeployedPipeline {
					build {
						id
					}
				}
			}
		}
	`, map[string]interface{}{
		"project": project,
		"branch":  branch,
		"build":   build,
		"slot":    slotId,
	}, &data); err != nil {
		return 0, 0, err
	}

	if len(data.Builds) == 0 {
		return 0, 0, fmt.Errorf("no build %s in branch %s of project %s", build, branch, project)
	}
	if data.SlotById == nil {
		return 0, 0, fmt.Errorf("no slot with ID %s", slotId)
	}
	if data.SlotById.LastDeployedPipeline == nil || data.SlotById.LastDeployedPipeline.Build == nil {
		return 0, 0, fmt.Errorf("nothing deployed in slot %s yet", slotId)
	}

	if from, err = strconv.Atoi(data.SlotById.LastDeployedPipeline.Build.Id); err != nil {
		return 0, 0, fmt.Errorf("invalid build ID %q returned by server", data.SlotById.LastDeployedPipeline.Build.Id)
	}
	if to, err = strconv.Atoi(data.Builds[0].Id); err != nil {
		return 0, 0, fmt.Errorf("invalid build ID %q returned by server", data.Builds[0].Id)
	}
	return from, to, nil
}

// sinceDeployedLine is the line of a commit: its short ID, its subject, its
// author, and its assistants with their markers.
func sinceDeployedLine(commit sinceDeployedCommit) string {
	subject, _, _ := strings.Cut(strings.TrimSpace(commit.Message), "\n")
	line := fmt.Sprintf("%s %s (%s)", commit.ShortId, strings.TrimSpace(subject), commit.Author)
	if len(commit.Assistants) == 0 {
		return line
	}
	assistants := make([]string, 0, len(commit.Assistants))
	for _, assistant := range commit.Assistants {
		assistants = append(assistants, assistant.Name+": "+strings.Join(assistant.Markers, ", "))
	}
	return line + " [" + strings.Join(assistants, "; ") + "]"
}

func init() {
	buildChangelogCmd.AddCommand(buildChangelogSinceDeployedCmd)

	buildChangelogSinceDeployedCmd.Flags().StringP("project", "p", "", "Name of the project (defaults to YONTRACK_PROJECT_NAME)")
	buildChangelogSinceDeployedCmd.Flags().StringP("branch", "b", "", "Name of the branch of the candidate build (defaults to YONTRACK_BRANCH_NAME)")
	buildChangelogSinceDeployedCmd.Flags().StringP("build", "n", "", "Name of the candidate build (defaults to YONTRACK_BUILD_NAME)")
	buildChangelogSinceDeployedCmd.Flags().String("slot", "", "ID of the slot whose last deployed build the change log starts from")
	buildChangelogSinceDeployedCmd.Flags().StringP("output", "o", "", "How to output the change log (json). Text by default.")
}
