package client

import (
	"yontrack/config"
)

func ValidateWithTests(
	cfg *config.Config,
	project string,
	branch string,
	build string,
	validation string,
	description string,
	runInfo *RunInfo,
	passed int,
	skipped int,
	failed int,
	status *string,
) (string, error) {

	// Mutation payload
	var payload struct {
		ValidateBuildWithTests struct {
			ValidationRun struct {
				Id string
			}
			Errors []struct {
				Message string
			}
		}
	}

	// Runs the mutation
	if err := GraphQLCall(cfg, `
			mutation ValidateBuildWithTests(
				$project: String!,
				$branch: String!,
				$build: String!,
				$validationStamp: String!,
				$description: String!,
				$runInfo: RunInfoInput,
				$passed: Int!,
				$skipped: Int!,
				$failed: Int!,
				$status: String
			) {
				validateBuildWithTests(input: {
					project: $project,
					branch: $branch,
					build: $build,
					validation: $validationStamp,
					description: $description,
					runInfo: $runInfo,
					passed: $passed,
					skipped: $skipped,
					failed: $failed,
					status: $status
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
		"passed":          passed,
		"skipped":         skipped,
		"failed":          failed,
		"status":          status,
	}, &payload); err != nil {
		return "", err
	}

	// Checks for errors
	if err := CheckDataErrors(payload.ValidateBuildWithTests.Errors); err != nil {
		return "", err
	}

	// ID of the created validation run
	return payload.ValidateBuildWithTests.ValidationRun.Id, nil
}
