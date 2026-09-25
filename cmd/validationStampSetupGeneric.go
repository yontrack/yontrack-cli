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
)

// validationStampSetupGenericCmd represents the validationStampSetupGeneric command
var validationStampSetupGenericCmd = &cobra.Command{
	Use:   "generic",
	Short: "Setup a validation stamp using a generic format",
	Long: `Setup a validation stamp using a generic format.

This is the same as 'yontrack vs setup', see 'yontrack vs setup --help'.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return setupValidationStamp(cmd)
	},
}

func init() {
	validationStampSetupCmd.AddCommand(validationStampSetupGenericCmd)

	addDataTypeFlags(validationStampSetupGenericCmd)
}
