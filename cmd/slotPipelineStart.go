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
	"os"
	"strings"
	"yontrack/client"
	config "yontrack/config"

	"github.com/spf13/cobra"
)

// slotPipelineStartCmd represents the slot pipeline start command
var slotPipelineStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the deployment of a build into a slot",
	Long: `Starts the deployment of a build into the slot of a project in an environment.

Identify the build either by its name:

    yontrack slot pipeline start --project my-project --environment production --build 1

or by its version, which is its release property:

    yontrack slot pipeline start --project my-project --environment production --version 1.0.0

Starting a pipeline is only the beginning of a deployment: the slot's own
workflows carry it the rest of the way. The pipeline ID is printed so that the
deployment can be followed afterwards.

The build must be eligible for the slot, or the command fails. An eligible build
is not always deployable yet, though - one which is not promoted yet, for
example. The pipeline is then created as a candidate, which cannot run until
the slot's admission rules are met. The command still succeeds, but prints a
warning on stderr listing the rules which are not met. To list the builds
which are deployable:

    yontrack slot builds --project my-project --environment production
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return slotPipelineStart(cmd)
	},
}

func slotPipelineStart(cmd *cobra.Command) error {
	project, err := cmd.Flags().GetString("project")
	if err != nil {
		return err
	}
	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		return err
	}
	name, err := cmd.Flags().GetString("build")
	if err != nil {
		return err
	}
	version, err := cmd.Flags().GetString("version")
	if err != nil {
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	if err := checkBuildSelector(name, version); err != nil {
		return err
	}

	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	found, err := getSlot(cfg, project, environment)
	if err != nil {
		return err
	}

	buildId, err := resolveBuildID(cfg, project, name, version)
	if err != nil {
		return err
	}

	pipeline, err := startSlotPipeline(cfg, found.Id, buildId)
	if err != nil {
		return err
	}

	// A candidate which cannot run yet is still a success, but it must not go
	// unnoticed. The warning goes to stderr, so that the output stays parseable.
	if warning := notDeployableWarning(pipeline); warning != "" {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), warning)
	}

	switch output {
	case "id":
		fmt.Println(pipeline.Id)
	case "json":
		bytes, err := json.MarshalIndent(pipeline, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", bytes)
	case "env":
		_, _ = fmt.Fprintf(os.Stdout, "export YONTRACK_PIPELINE_ID=%s\n", pipeline.Id)
		_, _ = fmt.Fprintf(os.Stdout, "export YONTRACK_PIPELINE_NUMBER=%d\n", pipeline.Number)
	default:
		return fmt.Errorf("unknown output format: %s", output)
	}

	return nil
}

// checkBuildSelector makes sure exactly one way of identifying the build was given.
func checkBuildSelector(name, version string) error {
	if name == "" && version == "" {
		return errors.New("one of --build or --version is required")
	}
	if name != "" && version != "" {
		return errors.New("--build and --version are mutually exclusive")
	}
	return nil
}

// notDeployableWarning is the warning to print when the pipeline was created as
// a candidate which cannot run yet, listing the admission rules which are not
// met. It is "" when the pipeline can run, or when the server does not tell.
func notDeployableWarning(pipeline *startedSlotPipeline) string {
	if pipeline.RunAction == nil || pipeline.RunAction.Ok == nil || *pipeline.RunAction.Ok {
		return ""
	}
	var reasons strings.Builder
	for _, rule := range pipeline.AdmissionRules {
		if rule.Overridden || (rule.Check.Ok != nil && *rule.Check.Ok) {
			continue
		}
		name := rule.AdmissionRuleConfig.Name
		if name == "" {
			name = rule.AdmissionRuleConfig.RuleId
		}
		_, _ = fmt.Fprintf(&reasons, "  - %s: %s\n", name, rule.Check.Reason)
	}
	if reasons.Len() == 0 {
		return fmt.Sprintf("Pipeline #%d created as a candidate — not deployable yet\n", pipeline.Number)
	}
	return fmt.Sprintf("Pipeline #%d created as a candidate — not deployable yet:\n%s", pipeline.Number, reasons.String())
}

// startSlotPipelineInput builds the StartSlotPipelineInput for the mutation.
// buildId is an Int in the schema, so it is passed as a number and never as a string.
func startSlotPipelineInput(slotId string, buildId int) map[string]interface{} {
	return map[string]interface{}{
		"slotId":  slotId,
		"buildId": buildId,
	}
}

// startSlotPipeline starts a pipeline for the build on the slot, and returns it.
func startSlotPipeline(cfg *config.Config, slotId string, buildId int) (*startedSlotPipeline, error) {
	var data struct {
		StartSlotPipeline struct {
			Pipeline *startedSlotPipeline
			Errors   []struct{ Message string }
		}
	}

	if err := client.GraphQLCall(cfg, `
		mutation StartSlotPipeline($input: StartSlotPipelineInput!) {
			startSlotPipeline(input: $input) {
				pipeline {
					id
					number
					runAction {
						ok
					}
					admissionRules {
						admissionRuleConfig {
							name
							ruleId
						}
						check {
							ok
							reason
						}
						overridden
					}
				}
				errors {
					message
				}
			}
		}
	`, map[string]interface{}{
		"input": startSlotPipelineInput(slotId, buildId),
	}, &data); err != nil {
		return nil, err
	}

	if err := client.CheckDataErrors(data.StartSlotPipeline.Errors); err != nil {
		return nil, err
	}

	// No pipeline and no error means nothing was started, whatever the HTTP status said.
	if data.StartSlotPipeline.Pipeline == nil {
		return nil, errors.New("no pipeline was started, and no error was reported either")
	}

	return data.StartSlotPipeline.Pipeline, nil
}

func init() {
	slotPipelineCmd.AddCommand(slotPipelineStartCmd)

	slotPipelineStartCmd.Flags().StringP("project", "p", "", "Name of the project")
	slotPipelineStartCmd.Flags().StringP("environment", "e", "", "Name of the environment")
	slotPipelineStartCmd.Flags().StringP("build", "b", "", "Name of the build to deploy")
	slotPipelineStartCmd.Flags().StringP("version", "v", "", "Version (release property) of the build to deploy")
	slotPipelineStartCmd.Flags().StringP("output", "o", "id", "How to display the pipeline (id, json, env)")

	_ = slotPipelineStartCmd.MarkFlagRequired("project")
	_ = slotPipelineStartCmd.MarkFlagRequired("environment")
}
