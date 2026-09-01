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
	"regexp"
	"strings"
	"yontrack/client"
	config "yontrack/config"

	"github.com/spf13/cobra"
)

// graphQLCmd represents the graphQL command
var graphQLCmd = &cobra.Command{
	Use:   "graphql",
	Short: "Performs a GraphQL command",
	Long: `Performs a GraphQL command.
	
For example:

    yontrack graphql --query 'query ProjectList($name: String!) { projects(name: $name) { id name branches { name } } }' --var name=ontrack

--var always sends a string. For a variable of any other type, pass the whole
variables object as JSON:

    yontrack graphql --query 'mutation Start($input: StartSlotPipelineInput!) { startSlotPipeline(input: $input) { errors { message } } }' --vars-json '{"input": {"slotId": "abc", "buildId": 11404}}'
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		query, err := cmd.Flags().GetString("query")
		if err != nil {
			return err
		}

		vars, err := cmd.Flags().GetStringSlice("var")
		if err != nil {
			return err
		}

		varsJson, err := cmd.Flags().GetString("vars-json")
		if err != nil {
			return err
		}

		variables, err := buildVariables(varsJson, vars)
		if err != nil {
			return err
		}

		cfg, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
		}

		var data interface{}

		if err := client.GraphQLCall(cfg, query, variables, &data); err != nil {
			return err
		}

		res, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(res))

		// OK
		return nil
	},
}

// buildVariables assembles the GraphQL variables from --vars-json and --var.
//
// --var can only ever express a string, so a query declaring a non-string
// variable - $buildId: Int! being the common case - cannot be driven by it: the
// server receives "11404" and rejects it. --vars-json takes the whole variables
// object as JSON instead, so every GraphQL type is expressible and no guessing
// about the intended type is needed.
//
// Both can be used together; --var entries are applied last and win, which
// makes it convenient to override a single string in an otherwise fixed object.
func buildVariables(varsJson string, vars []string) (map[string]interface{}, error) {
	variables := map[string]interface{}{}

	if varsJson != "" {
		decoder := json.NewDecoder(strings.NewReader(varsJson))
		// Keeps numbers exactly as written, rather than routing them through
		// float64 and risking a loss of precision on large integers.
		decoder.UseNumber()
		var parsed interface{}
		if err := decoder.Decode(&parsed); err != nil {
			return nil, fmt.Errorf("--vars-json is not valid JSON: %w", err)
		}
		object, ok := parsed.(map[string]interface{})
		if !ok {
			return nil, errors.New("--vars-json must be a JSON object")
		}
		for name, value := range object {
			variables[name] = value
		}
	}

	for _, token := range vars {
		name, value, err := parseVar(token)
		if err != nil {
			return nil, err
		}
		variables[name] = value
	}

	return variables, nil
}

func parseVar(token string) (string, string, error) {
	re := regexp.MustCompile(`^(.+)=(.*)$`)
	match := re.FindStringSubmatch(token)
	if match == nil {
		return "", "", errors.New("Variable " + token + " must match name=value")
	}

	name := match[1]
	value := match[2]

	return name, value, nil
}

func init() {
	rootCmd.AddCommand(graphQLCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// graphQLCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// graphQLCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	graphQLCmd.Flags().StringP("query", "q", "", "GraphQL query")
	graphQLCmd.Flags().StringSliceP("var", "v", []string{}, "GraphQL variable, in the form name=value. Always sent as a string; use --vars-json for any other type.")
	graphQLCmd.Flags().String("vars-json", "", "GraphQL variables as a JSON object, for variables which are not strings")

	graphQLCmd.MarkFlagRequired("query")
}
