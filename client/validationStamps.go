package client

import (
	"encoding/json"

	config "yontrack/config"
)

// SetupValidationStamp creates or updates a validation stamp. The data type
// configuration, if any, is sent as a JSON variable, never spliced into the
// query; nil sends null.
func SetupValidationStamp(
	cfg *config.Config,
	project string,
	branch string,
	validation string,
	description string,
	dataType string,
	dataTypeConfig json.RawMessage,
) error {
	var data struct {
		SetupValidationStamp struct {
			Errors []struct {
				Message string
			}
		}
	}
	if err := GraphQLCall(cfg, `
		mutation SetupValidationStamp(
			$project: String!,
			$branch: String!,
			$validation: String!,
			$description: String,
			$dataType: String,
			$dataTypeConfig: JSON
		) {
			setupValidationStamp(input: {
				project: $project,
				branch: $branch,
				validation: $validation,
				description: $description,
				dataType: $dataType,
				dataTypeConfig: $dataTypeConfig
			}) {
				errors {
					message
				}
			}
		}
	`, map[string]interface{}{
		"project":        project,
		"branch":         branch,
		"validation":     validation,
		"description":    description,
		"dataType":       dataType,
		"dataTypeConfig": dataTypeConfig,
	}, &data); err != nil {
		return err
	}

	// Checks for errors
	if err := CheckDataErrors(data.SetupValidationStamp.Errors); err != nil {
		return err
	}

	// OK
	return nil
}
