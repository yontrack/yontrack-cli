package utils

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

func GetProjectFlag(cmd *cobra.Command) (string, error) {
	project, err := cmd.Flags().GetString("project")
	if err != nil {
		return "", err
	}
	if project == "" {
		project = os.Getenv("YONTRACK_PROJECT_NAME")
	}
	if project == "" {
		return "", errors.New("project is required (use --project flag or YONTRACK_PROJECT_NAME environment variable)")
	} else {
		return project, nil
	}
}

func GetBranchFlag(cmd *cobra.Command, ignoreEmptyBranch bool, normalizeBranch bool) (string, error) {
	branch, err := cmd.Flags().GetString("branch")
	if err != nil {
		return "", err
	}
	if branch == "" {
		branch = os.Getenv("YONTRACK_BRANCH_NAME")
	}
	if branch == "" && !ignoreEmptyBranch {
		return "", errors.New("branch is required (use --branch flag or YONTRACK_BRANCH_NAME environment variable)")
	} else if normalizeBranch {
		return NormalizeBranchName(branch), nil
	} else {
		return branch, nil
	}
}

func GetProjectBranchFlags(cmd *cobra.Command, ignoreEmpty bool, normalizeBranch bool) (string, string, error) {
	project, err := GetProjectFlag(cmd)
	if err != nil {
		return "", "", err
	}
	branch, err := GetBranchFlag(cmd, ignoreEmpty, normalizeBranch)
	if err != nil {
		return "", "", err
	}
	return project, branch, nil
}

func GetBuildFlag(cmd *cobra.Command) (string, error) {
	build, err := cmd.Flags().GetString("build")
	if err != nil {
		return "", err
	}
	if build == "" {
		build = os.Getenv("YONTRACK_BUILD_NAME")
	}
	if build == "" {
		return "", errors.New("build is required (use --build flag or YONTRACK_BUILD_NAME environment variable)")
	} else {
		return build, nil
	}
}

// GetPipelineFlag reads the ID of a slot pipeline, from --pipeline or else from
// YONTRACK_PIPELINE_ID, which 'slot pipeline start --output env' exports.
func GetPipelineFlag(cmd *cobra.Command) (string, error) {
	pipeline, err := cmd.Flags().GetString("pipeline")
	if err != nil {
		return "", err
	}
	if pipeline == "" {
		pipeline = os.Getenv("YONTRACK_PIPELINE_ID")
	}
	if pipeline == "" {
		return "", errors.New("pipeline is required (use --pipeline flag or YONTRACK_PIPELINE_ID environment variable)")
	}
	return pipeline, nil
}

func GetBuildIdFromEnv() (int, error) {
	buildIdStr := os.Getenv("YONTRACK_BUILD_ID")
	if buildIdStr == "" {
		return 0, errors.New("use YONTRACK_BUILD_ID environment variable to set the build ID")
	}
	buildId, err := strconv.Atoi(buildIdStr)
	if err != nil {
		return 0, err
	}
	return buildId, nil
}

func GetProjectBranchBuildFlags(cmd *cobra.Command, ignoreEmptyBranch bool, normalizeBranch bool) (string, string, string, error) {
	project, branch, err := GetProjectBranchFlags(cmd, ignoreEmptyBranch, normalizeBranch)
	if err != nil {
		return "", "", "", err
	}
	build, err := GetBuildFlag(cmd)
	if err != nil {
		return "", "", "", err
	}
	return project, branch, build, nil
}

// dateFlagLayouts are the formats accepted by --date. A date/time without an
// offset is taken as UTC, as Yontrack does.
var dateFlagLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// GetDateFlag reads --date, the date/time an action is backdated to, and
// returns it as Yontrack expects a LocalDateTime: in UTC, without an offset.
// It is "" when --date is not set, so that Yontrack takes the current time.
func GetDateFlag(cmd *cobra.Command) (string, error) {
	value, err := cmd.Flags().GetString("date")
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", nil
	}
	for _, layout := range dateFlagLayouts {
		if date, err := time.Parse(layout, value); err == nil {
			return date.UTC().Format("2006-01-02T15:04:05"), nil
		}
	}
	return "", fmt.Errorf("invalid --date %s: expected 2006-01-02, 2006-01-02T15:04:05 (UTC) or 2006-01-02T15:04:05Z07:00", value)
}
