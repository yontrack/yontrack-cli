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
	"fmt"

	"github.com/spf13/cobra"

	client "yontrack/client"
	config "yontrack/config"
)

var versionCli bool
var versionOntrack bool

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Displays version information",
	Long: `Displays the version of the CLI and the remote Yontrack instance:

    yontrack version

To display only the CLI version, run:

	yontrack version --cli

To display only the Yontrack version, run:

	yontrack version --ontrack
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return version()
	},
}

func version() error {
	both := (versionCli && versionOntrack) || (!versionCli && !versionOntrack)
	var ontrackVersion string
	var ontrackURL string
	if both || versionOntrack {
		cfg, err := config.GetSelectedConfiguration()
		if err != nil {
			return err
		}
		ontrackURL = cfg.URL

		var data struct {
			Info struct {
				Version struct {
					Display string
				}
			}
		}

		if err := client.GraphQLCall(cfg, `
			{
				info {
					version {
						display
					}
				}
			}
		`, map[string]interface{}{}, &data); err != nil {
			return err
		}

		ontrackVersion = data.Info.Version.Display
	}
	if both {
		fmt.Printf("CLI Version %s\n", config.Version)
		fmt.Printf("Yontrack URL %s\n", ontrackURL)
		fmt.Printf("Yontrack Version %s\n", ontrackVersion)
	} else if versionCli {
		fmt.Println(config.Version)
	} else if versionOntrack {
		fmt.Println(ontrackVersion)
	} else {
		return errors.New("No version was asked")
	}
	return nil
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	versionCmd.Flags().BoolVarP(&versionCli, "cli", "c", false, "Displays the CLI version")
	versionCmd.Flags().BoolVarP(&versionOntrack, "ontrack", "o", false, "Displays the Yontrack version for the current configuration")
}
