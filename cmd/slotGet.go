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
	"fmt"
	"os"
	config "yontrack/config"

	"github.com/spf13/cobra"
)

// slotGetCmd represents the slot get command
var slotGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Gets a deployment slot and what it last deployed",
	Long: `Gets the deployment slot of a project in an environment, together with the build
it last deployed.

    yontrack slot get --project my-project --environment production

By default, the slot ID is printed on its own, which is what most scripts want:

    yontrack slot get --project my-project --environment production --output id

The last deployed build is what tells you whether a deployment is needed at all.
To get everything, use the JSON output:

    yontrack slot get --project my-project --environment production --output json
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return slotGet(cmd)
	},
}

func slotGet(cmd *cobra.Command) error {
	project, err := cmd.Flags().GetString("project")
	if err != nil {
		return err
	}
	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	cfg, err := config.GetSelectedConfiguration()
	if err != nil {
		return err
	}

	slot, err := getSlot(cfg, project, environment)
	if err != nil {
		return err
	}

	switch output {
	case "id":
		fmt.Println(slot.Id)
	case "json":
		bytes, err := json.MarshalIndent(slot, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", bytes)
	case "env":
		_, _ = fmt.Fprintf(os.Stdout, "export YONTRACK_SLOT_ID=%s\n", slot.Id)
		_, _ = fmt.Fprintf(os.Stdout, "export YONTRACK_SLOT_DEPLOYED_BUILD_ID=%s\n", slot.DeployedBuildId())
		_, _ = fmt.Fprintf(os.Stdout, "export YONTRACK_SLOT_DEPLOYED_BUILD_NAME=%s\n", slot.DeployedBuildName())
	default:
		return fmt.Errorf("unknown output format: %s", output)
	}

	return nil
}

func init() {
	slotCmd.AddCommand(slotGetCmd)

	slotGetCmd.Flags().StringP("project", "p", "", "Name of the project")
	slotGetCmd.Flags().StringP("environment", "e", "", "Name of the environment")
	slotGetCmd.Flags().StringP("output", "o", "id", "How to display the slot (id, json, env)")

	_ = slotGetCmd.MarkFlagRequired("project")
	_ = slotGetCmd.MarkFlagRequired("environment")
}
