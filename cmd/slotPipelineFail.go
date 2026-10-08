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
	"errors"
	"yontrack/client"
	config "yontrack/config"
	"yontrack/utils"

	"github.com/spf13/cobra"
)

// slotPipelineFailCmd represents the slot pipeline fail command
var slotPipelineFailCmd = &cobra.Command{
	Use:   "fail",
	Short: "Marks a running deployment as failed",
	Long: `Marks a running deployment as failed. Requires Yontrack 6.0.

The pipeline is identified by its ID, as printed by 'slot pipeline start':

    yontrack slot pipeline fail --pipeline 2957ff78-... --message "Smoke tests failed"

Without --pipeline, the ID is taken from YONTRACK_PIPELINE_ID, which
'slot pipeline start --output env' exports:

    eval "$(yontrack slot pipeline start --project my-project --environment production --build 1 --output env)"
    ...
    yontrack slot pipeline fail --message "Smoke tests failed"

Only a running deployment can fail: a candidate which never started is
cancelled instead. A failed deployment does not change what the slot runs.

To record a failure which happened in the past, --date backdates it:

    yontrack slot pipeline fail --pipeline 2957ff78-... --date 2026-09-15T14:45:00Z

It cannot be in the future, before the creation of the build, nor before the
previous change of the pipeline. A date/time without an offset is in UTC.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return slotPipelineFail(cmd)
	},
}

func slotPipelineFail(cmd *cobra.Command) error {
	pipelineId, err := utils.GetPipelineFlag(cmd)
	if err != nil {
		return err
	}
	message, err := cmd.Flags().GetString("message")
	if err != nil {
		return err
	}
	dateTime, err := utils.GetDateFlag(cmd)
	if err != nil {
		return err
	}

	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	var data struct {
		FailSlotPipeline struct {
			FailStatus *struct {
				Ok      *bool
				Message string
			}
			Errors []struct{ Message string }
		}
	}

	if err := client.GraphQLCall(cfg, `
		mutation FailSlotPipeline($input: FailSlotPipelineInput!) {
			failSlotPipeline(input: $input) {
				failStatus {
					ok
					message
				}
				errors {
					message
				}
			}
		}
	`, map[string]interface{}{
		"input": failSlotPipelineInput(pipelineId, message, dateTime),
	}, &data); err != nil {
		return err
	}

	if err := client.CheckDataErrors(data.FailSlotPipeline.Errors); err != nil {
		return err
	}

	// A pipeline which cannot fail - one which is not running - is refused
	// through the status of the action, not through the errors.
	status := data.FailSlotPipeline.FailStatus
	if status == nil {
		return errors.New("the pipeline was not marked as failed, and no error was reported either")
	}
	if status.Ok == nil || !*status.Ok {
		if status.Message == "" {
			return errors.New("the pipeline was not marked as failed")
		}
		return errors.New(status.Message)
	}
	return nil
}

// failSlotPipelineInput builds the FailSlotPipelineInput for the mutation. An
// empty message or date/time is left out, for Yontrack to use its defaults: a
// generic message, and the current time.
func failSlotPipelineInput(pipelineId, message, dateTime string) map[string]interface{} {
	input := map[string]interface{}{
		"pipelineId": pipelineId,
	}
	if message != "" {
		input["message"] = message
	}
	if dateTime != "" {
		input["dateTime"] = dateTime
	}
	return input
}

func init() {
	slotPipelineCmd.AddCommand(slotPipelineFailCmd)

	slotPipelineFailCmd.Flags().String("pipeline", "", "ID of the running pipeline to mark as failed (defaults to YONTRACK_PIPELINE_ID)")
	slotPipelineFailCmd.Flags().StringP("message", "m", "", "Why the deployment failed, recorded in the pipeline's history")
	slotPipelineFailCmd.Flags().String("date", "", "Date/time the deployment failed, to backdate it (2006-01-02T15:04:05Z07:00, UTC when no offset is given)")
}
