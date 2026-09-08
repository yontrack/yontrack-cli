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
	"github.com/spf13/cobra"

	config "yontrack/config"
)

// authentication
var username string
var password string
var token string
var connectionRetry config.ConnectionRetry

// configCreateCmd represents the configCreate command
var configCreateCmd = &cobra.Command{
	Use:   "create NAME URL",
	Short: "Creates a new configuration",
	Long: `To create a 'local' configuration to connect to a local instance of Yontrack 
using a username and a password:
	
	yontrack config create local http://localhost:8080 --username <username> --password <password>
		
or to create a 'prod' configuration using a token:
	
	yontrack config create prod https://ontrack.nemerosa.net --token <token>
`,
	Args: cobra.ExactValidArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return createConfig(cmd, args)
	},
}

func createConfig(cmd *cobra.Command, args []string) error {
	// Arguments have already been validated by ExactValidArgs(2) in the command definition
	name := args[0]
	url := args[1]

	// Override parameter
	override, err := cmd.Flags().GetBool("override")
	if err != nil {
		return err
	}

	// Creates the configuration
	var cfg = config.Config{
		Name:            name,
		URL:             url,
		Username:        username,
		Password:        password,
		Token:           token,
		ConnectionRetry: connectionRetry,
	}

	// Adds this configuration to the file
	// and sets as default
	if err := config.AddConfiguration(cfg, override); err != nil {
		return err
	}

	// OK
	return nil
}

func init() {
	configCmd.AddCommand(configCreateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCreateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:

	// Authentication flags

	configCreateCmd.Flags().StringVarP(&username, "username", "u", "", "Username for basic authentication")
	configCreateCmd.Flags().StringVarP(&password, "password", "p", "", "Password for basic authentication")
	configCreateCmd.Flags().StringVarP(&token, "token", "t", "", "Token based authentication (if defined, takes priority over username/password authentication)")
	configCreateCmd.Flags().BoolP("override", "o", false, "Overrides the configuration if it already exists")
	configCreateCmd.Flags().IntVarP(&connectionRetry.MaxWaitTimeSec, "conn-retry-wait", "", 2, "Max connection retry wait time between attempts in seconds")
	configCreateCmd.Flags().IntVarP(&connectionRetry.MaxCount, "conn-retry-count", "", 5, "Max connection retry attempts")
}
