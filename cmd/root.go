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
	"yontrack/config"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "yontrack",
	Short: "Yontrack Command Line Interface",
	Long: `The Yontrack CLI allows you to communicate with a Yontrack server.

First, you need to configure the connection to Yontrack. For example:

	yontrack config create local http://localhost:8080 --username <user> --password <password>
	
Or, using a token:

	yontrack config create local http://localhost:8080 --token <token>
	
Note that you can create several configurations and manage them using

	yontrack config
	
Examples of usages:

* To get the list of projects:

	yontrack project list
	
* To setup a project and a branch in an idempotent way:

	yontrack branch setup --project my-project --branch release/1.0

* To create a validation stamp in an idempotent way:

	yontrack validation setup --project my-project --branch release/1.0 --validation TESTS
	
* To create a new build on a branch and project:

	yontrack build create --project my-project --branch release/1.0 --build 123
	
* To create a validation run on an existing build:

	yontrack build validate --project my-project --branch release/1.0 --build 123 --validation TESTS --status PASSED
	`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) {},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&config.ConfigFilePath, "config", "./.yontrack-config.yaml", "Configuration file path.")

	rootCmd.PersistentFlags().BoolVar(&config.GraphQLLogging, "graphql-log", false, "Enable traces on the GraphQL calls.")
}
