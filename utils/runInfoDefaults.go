package utils

import (
	"fmt"
	"os"
)

// BitbucketPipelineSourceType is the run info source type used for a build or a
// validation run created from a Bitbucket Pipelines pipeline. The exact string
// matters: it must match the RunInfoSourceTypeIcon registered in Yontrack
// (yontrack/yontrack#1760) for the pipeline icon & link to be rendered.
const BitbucketPipelineSourceType = "bitbucket-pipeline"

// GitLabPipelineSourceType is the run info source type used for a build or a
// validation run created from a GitLab CI pipeline. As for Bitbucket, the exact
// string matters: it must match the RunInfoSourceTypeIcon registered in
// Yontrack for the gitlab-ci CI engine.
const GitLabPipelineSourceType = "gitlab-pipeline"

// RunInfoDefaults holds the run info fields which can be guessed from the
// environment of the CI engine the CLI runs in. The run time is never guessed:
// only the command itself knows how long the run took.
type RunInfoDefaults struct {
	SourceType  string
	SourceURI   string
	TriggerType string
	TriggerData string
}

// GetRunInfoDefaults returns the run info fields which can be defaulted from
// the CI environment. All CI engine detections are gathered here so that new
// engines (GitHub Actions, ...) can be added beside the existing ones. An empty
// RunInfoDefaults is returned when no known CI engine is detected.
//
// The engines are tried in a fixed order and the first one which recognises its
// environment wins. Two engines cannot really run the same job, but their
// variables can both be present by accident - a user-defined BITBUCKET_* CI
// variable in a GitLab job, for instance. Ordering by seniority, and therefore
// appending each new engine at the end, keeps that accident from changing what
// the users of the engines already supported were getting.
func GetRunInfoDefaults() RunInfoDefaults {
	if defaults, ok := bitbucketPipelinesRunInfoDefaults(); ok {
		return defaults
	}
	if defaults, ok := gitLabCIRunInfoDefaults(); ok {
		return defaults
	}
	return RunInfoDefaults{}
}

// bitbucketPipelinesRunInfoDefaults detects Bitbucket Pipelines and returns the
// run info it can fill in. BITBUCKET_BUILD_NUMBER is the marker: it is always
// set by Bitbucket Pipelines and is needed to build the pipeline URL.
func bitbucketPipelinesRunInfoDefaults() (RunInfoDefaults, bool) {
	buildNumber := os.Getenv("BITBUCKET_BUILD_NUMBER")
	if buildNumber == "" {
		return RunInfoDefaults{}, false
	}
	defaults := RunInfoDefaults{
		SourceType:  BitbucketPipelineSourceType,
		TriggerType: "commit",
		TriggerData: os.Getenv("BITBUCKET_COMMIT"),
	}
	// The pipeline URL needs both the workspace and the repository slug: rather
	// than sending a broken link to Yontrack, no source URI is set when either
	// one is missing.
	workspace := os.Getenv("BITBUCKET_WORKSPACE")
	repository := os.Getenv("BITBUCKET_REPO_SLUG")
	if workspace != "" && repository != "" {
		defaults.SourceURI = fmt.Sprintf(
			"https://bitbucket.org/%s/%s/pipelines/results/%s",
			workspace,
			repository,
			buildNumber,
		)
	}
	return defaults, true
}

// gitLabCIRunInfoDefaults detects GitLab CI and returns the run info it can
// fill in. GITLAB_CI is the marker: GitLab sets it to "true" in every job, and
// only there. Unlike the Bitbucket marker it carries no data, so the values are
// read from the CI_* variables, each of them optional: an empty CI_PIPELINE_URL
// simply leaves the source URI unset rather than sending a broken link to
// Yontrack. CI_PIPELINE_URL is job-only, which is enough here: the CLI runs
// inside a job.
func gitLabCIRunInfoDefaults() (RunInfoDefaults, bool) {
	if os.Getenv("GITLAB_CI") != "true" {
		return RunInfoDefaults{}, false
	}
	return RunInfoDefaults{
		SourceType:  GitLabPipelineSourceType,
		SourceURI:   os.Getenv("CI_PIPELINE_URL"),
		TriggerType: "commit",
		TriggerData: os.Getenv("CI_COMMIT_SHA"),
	}, true
}
