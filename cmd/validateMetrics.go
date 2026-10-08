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
	"regexp"
	"strconv"
	"strings"
	"yontrack/utils"

	"github.com/spf13/cobra"

	client "yontrack/client"
	config "yontrack/config"
)

// validateMetricsCmd represents the validateMetrics command
var validateMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Validation with metrics data",
	Long: `Validation with metrics data.

For example:

    yontrack validate -p PROJECT -b BRANCH -n BUILD -v VALIDATION metrics --metric name1=value1 --metric name2=value2

An alternative syntax is:

	yontrack validate -p PROJECT -b BRANCH -n BUILD -v VALIDATION metrics --metrics name1=value1,name2=value2
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		project, branch, build, err := utils.GetProjectBranchBuildFlags(cmd, false, true)
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

		runInfo, err := GetRunInfo(cmd)
		if err != nil {
			return err
		}

		// List of metrics
		var metricList = []metric{}

		// Adding from the `metric` flags
		metricArgs, err := cmd.Flags().GetStringSlice("metric")
		for _, value := range metricArgs {
			name, metricValue, err := parseMetric(value)
			if err != nil {
				return err
			}
			metricList = append(metricList, metric{
				Name:  name,
				Value: metricValue,
			})
		}

		// Adding from the `metrics` flag
		metricsArg, err := cmd.Flags().GetString("metrics")
		if err != nil {
			return err
		}
		if metricsArg != "" {
			metricsArgTokens := strings.Split(metricsArg, ",")
			for _, value := range metricsArgTokens {
				name, metricValue, err := parseMetric(value)
				if err != nil {
					return err
				}
				metricList = append(metricList, metric{
					Name:  name,
					Value: metricValue,
				})
			}
		}

		// Evidence
		evidence, err := getValidateEvidence(cmd)
		if err != nil {
			return err
		}

		// Get the configuration
		cfg, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
		}

		// Mutation payload
		var payload struct {
			ValidateBuildWithMetrics struct {
				ValidationRun struct {
					Id string
				}
				Errors []struct {
					Message string
				}
			}
		}

		// Runs the mutation
		if err := client.GraphQLCall(cfg, `
			mutation ValidateBuildWithMetrics(
				$project: String!,
				$branch: String!,
				$build: String!,
				$validationStamp: String!,
				$description: String!,
				$runInfo: RunInfoInput,
				$metrics: [MetricsEntryInput!]!
			) {
				validateBuildWithMetrics(input: {
					project: $project,
					branch: $branch,
					build: $build,
					validation: $validationStamp,
					description: $description,
					runInfo: $runInfo,
					metrics: $metrics
				}) {
					validationRun {
						id
					}
					errors {
						message
					}
				}
			}
		`, map[string]interface{}{
			"project":         project,
			"branch":          branch,
			"build":           build,
			"validationStamp": validation,
			"description":     description,
			"runInfo":         runInfo,
			"metrics":         metricList,
		}, &payload); err != nil {
			return err
		}

		// Checks for errors
		if err := client.CheckDataErrors(payload.ValidateBuildWithMetrics.Errors); err != nil {
			return err
		}

		// Evidence
		return evidence.attach(cmd, cfg, payload.ValidateBuildWithMetrics.ValidationRun.Id)
	},
}

func parseMetric(value string) (string, float64, error) {
	re := regexp.MustCompile(`^(.+)=(\d+(\.\d+)?)$`)
	match := re.FindStringSubmatch(value)
	if match == nil {
		return "", 0, errors.New("Metric " + value + " must match name=number[.number]")
	}

	name := match[1]
	metric, err := strconv.ParseFloat(match[2], 64)
	if err != nil {
		return "", 0, err
	}

	return name, metric, nil
}

func init() {
	validateCmd.AddCommand(validateMetricsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// validateMetricsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// validateMetricsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	validateMetricsCmd.Flags().StringSliceP("metric", "m", []string{}, "List of metric, each value being provided like 'name=value'")
	validateMetricsCmd.Flags().String("metrics", "", "Comma-separated list of metric, each value being provided like 'name=value'")

	// Run info arguments
	InitRunInfoCommandFlags(validateMetricsCmd)
}

type metric struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}
