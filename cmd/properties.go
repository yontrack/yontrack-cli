package cmd

import (
	"bytes"
	"strings"
	"text/template"
	client "yontrack/client"
	config "yontrack/config"
)

// SetProperty sets a property on any given entity using its type name and its value as a JSON string
func SetProperty(entityType string, entityNames map[string]string, typeName string, value string) error {
	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	tmpl, err := template.New("mutation").Parse(`
		mutation SetProperty($propertyName: String!, $propertyValue: JSON) {
			set{{.EntityTypeName}}Property(input: { {{.EntityInput}}, property: $propertyName, value: $propertyValue }) {
				errors {
					message
				}
			}
		}
	`)
	if err != nil {
		return err
	}

	var queryTmplInput = setPropertyQueryTmplInput{
		EntityTypeName: strings.Title(entityType),
		EntityTypeVar:  entityType,
		EntityInput:    entityInput(entityNames),
	}
	var query bytes.Buffer
	if err := tmpl.Execute(&query, queryTmplInput); err != nil {
		return err
	}

	// nodeName --> errors --> []error
	var data map[string]setPropertyPayload
	if err := client.GraphQLCall(
		cfg,
		query.String(),
		map[string]interface{}{
			"propertyName":  typeName,
			"propertyValue": value,
		},
		&data); err != nil {
		return err
	}

	// Check for errors
	var nodeName string = "set" + queryTmplInput.EntityTypeName + "Property"
	var errors = data[nodeName].Errors
	if err := client.CheckDataErrors(errors); err != nil {
		return err
	}

	// OK
	return nil
}

func entityInput(entityNames map[string]string) string {
	var vars []string
	for name, value := range entityNames {
		vars = append(vars, name+`: "`+value+`"`)
	}
	return strings.Join(vars, ", ")
}

// PropertyMapping Mapping of properties short names x entity ==> FQCN of the property in Yontrack
var PropertyMapping = map[string]map[string]string{
	"project": {
		"gitHub": "net.nemerosa.ontrack.extension.github.property.GitHubProjectConfigurationPropertyType",
	},
	"branch": {
		"git": "net.nemerosa.ontrack.extension.git.property.GitBranchConfigurationPropertyType",
	},
	"build": {
		"git": "net.nemerosa.ontrack.extension.git.property.GitCommitPropertyType",
	},
}

type setPropertyPayload struct {
	Errors []struct {
		Message string
	}
}

type setPropertyQueryTmplInput struct {
	EntityTypeName    string
	EntityTypeVar     string
	EntityInput       string
	PropertyTypeName  string
	PropertyTypeValue string
}

type Property struct {
	fields []PropertyField
}

type PropertyField struct {
	name     string
	required bool
	field    int
}

const (
	StringField int = iota
	IntField
	BoolField
)
