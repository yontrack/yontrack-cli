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

// auditTrailCmd represents the audit-trail command
var auditTrailCmd = &cobra.Command{
	Use:   "audit-trail",
	Short: "Audit trails of the builds",
	Long: `Audit trails of the builds.

Yontrack 6.0 records the story of every build in a trail: an append-only list of
entries, each one chained to the one before it by its SHA-256 hash and endorsed
by the Ed25519 key of the instance. The trail of a build is exported as JSON
from its "Audit trail" page.

To verify an export offline, without trusting the server it comes from:

    yontrack audit-trail verify audit-trail-payments-release-2.4-2.4.7.json
`,
}

func init() {
	rootCmd.AddCommand(auditTrailCmd)
}
