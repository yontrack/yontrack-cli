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
	"fmt"
	client "yontrack/client"
	config "yontrack/config"

	"github.com/spf13/cobra"
)

var showID bool

// projectListCmd represents the projectList command
var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "Displays the list of projects",
	Long: `Displays the list of projects.

	yontrack project list

By default, only the names are displayed. You can display the ID instead:

	yontrack project list --show-id
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return projectList()
	},
}

func projectList() error {
	config, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	data := new(projectListResponse)
	if err := client.GraphQLCall(config, `{ projects { id name }}`, map[string]interface{}{}, &data); err != nil {
		return err
	}

	for _, project := range data.Projects {
		if showID {
			fmt.Println(project.ID)
		} else {
			fmt.Println(project.Name)
		}
	}

	return nil
}

func init() {
	projectCmd.AddCommand(projectListCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// projectListCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	projectListCmd.Flags().BoolVarP(&showID, "show-id", "i", false, "Displays the ID instead of the name")
}

type projectListResponse struct {
	Projects []project
}

type project struct {
	ID   string
	Name string
}
