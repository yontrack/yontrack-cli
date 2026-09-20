package cmd

import (
	client "yontrack/client"
	"yontrack/utils"

	"github.com/spf13/cobra"
)

func InitRunInfoCommandFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("source-type", "", "Run info source type. Defaults to \"bitbucket-pipeline\" in Bitbucket Pipelines.")
	cmd.PersistentFlags().String("source-uri", "", "Run info source URI. Defaults to the pipeline URL in Bitbucket Pipelines.")
	cmd.PersistentFlags().String("trigger-type", "", "Run info trigger type. Defaults to \"commit\" in Bitbucket Pipelines.")
	cmd.PersistentFlags().String("trigger-data", "", "Run info trigger data. Defaults to $BITBUCKET_COMMIT in Bitbucket Pipelines.")
	cmd.PersistentFlags().Int("run-time", 0, "Run info run time (in seconds)")
}

func GetRunInfo(cmd *cobra.Command) (*client.RunInfo, error) {
	sourceType, err := cmd.Flags().GetString("source-type")
	if err != nil {
		return nil, err
	}
	sourceURI, err := cmd.Flags().GetString("source-uri")
	if err != nil {
		return nil, err
	}
	triggerType, err := cmd.Flags().GetString("trigger-type")
	if err != nil {
		return nil, err
	}
	triggerData, err := cmd.Flags().GetString("trigger-data")
	if err != nil {
		return nil, err
	}
	runTime, err := cmd.Flags().GetInt("run-time")
	if err != nil {
		return nil, err
	}

	// Fields not given on the command line are filled in from the CI
	// environment (Bitbucket Pipelines, ...) when one is detected.
	info := applyRunInfoDefaults(client.RunInfo{
		SourceType:  sourceType,
		SourceURI:   sourceURI,
		TriggerType: triggerType,
		TriggerData: triggerData,
		RunTime:     runTime,
	}, utils.GetRunInfoDefaults())

	if isRunInfoEmpty(info) {
		return nil, nil
	}
	return &info, nil
}

// applyRunInfoDefaults completes the run info given on the command line with
// the values guessed from the CI environment. Explicit flags always win.
func applyRunInfoDefaults(info client.RunInfo, defaults utils.RunInfoDefaults) client.RunInfo {
	if info.SourceType == "" {
		info.SourceType = defaults.SourceType
	}
	if info.SourceURI == "" {
		info.SourceURI = defaults.SourceURI
	}
	if info.TriggerType == "" {
		info.TriggerType = defaults.TriggerType
	}
	if info.TriggerData == "" {
		info.TriggerData = defaults.TriggerData
	}
	return info
}

// isRunInfoEmpty returns true when there is nothing to send to Yontrack.
func isRunInfoEmpty(info client.RunInfo) bool {
	return info.SourceType == "" &&
		info.SourceURI == "" &&
		info.TriggerType == "" &&
		info.TriggerData == "" &&
		info.RunTime == 0
}
