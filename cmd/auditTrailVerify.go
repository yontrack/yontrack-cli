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
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"yontrack/audittrail"

	"github.com/spf13/cobra"
)

// auditTrailVerifyCmd represents the audit-trail verify command
var auditTrailVerifyCmd = &cobra.Command{
	Use:   "verify FILE",
	Short: "Verifies the JSON export of an audit trail, offline",
	Long: `Verifies the JSON export of the audit trail of a build, offline.

    yontrack audit-trail verify audit-trail-payments-release-2.4-2.4.7.json

Nothing is asked from Yontrack: the export holds the entries, their
endorsements and the public keys of the instance. For each entry, in order:

- its seq is its position, from 1;
- its hash is the SHA-256 of the RFC 8785 canonical JSON of its envelope;
- it is chained to the entry before it by its previous hash;
- the first entry opens the trail of the build the export names;
- each endorsement is the Ed25519 signature of its hash by a key of the export.

The command prints a report, and fails when the chain is broken or an
endorsement is invalid. A partial trail - one opened on a build older than its
trail - and entries written while the instance key was not provisioned are
reported, but do not fail it.

Requires an export of Yontrack 6.0.
`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return auditTrailVerify(cmd, args[0])
	},
}

func auditTrailVerify(cmd *cobra.Command, file string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("cannot read the export: %w", err)
	}
	export, err := audittrail.ReadExport(content)
	if err != nil {
		return err
	}

	verification := audittrail.Verify(export)
	writeAuditTrailReport(cmd.OutOrStdout(), export, verification)

	if verification.Broken() {
		var reasons []string
		if !verification.ChainIntact {
			reasons = append(reasons, fmt.Sprintf("its chain is broken at seq %d", verification.FirstBrokenSeq))
		}
		if !verification.EndorsementsValid {
			reasons = append(reasons, fmt.Sprintf("an endorsement is invalid at seq %d", verification.FirstInvalidEndorsementSeq))
		}
		return errors.New("the trail is broken: " + strings.Join(reasons, ", and "))
	}
	return nil
}

// writeAuditTrailReport writes the verification of an export for a human.
func writeAuditTrailReport(out io.Writer, export *audittrail.Export, verification audittrail.Verification) {
	build := export.Build
	_, _ = fmt.Fprintf(out, "Trail of build %s / %s / %s (ID %d), exported at %s\n",
		build.Project, build.Branch, build.Name, build.Id, export.ExportedAt)
	_, _ = fmt.Fprintf(out, "Entries:      %d\n", len(export.Entries))

	var keys []string
	for _, key := range export.Keys {
		keys = append(keys, fmt.Sprintf("%s (%s)", key.KeyId, key.Algorithm))
	}
	if len(keys) == 0 {
		keys = []string{"none"}
	}
	_, _ = fmt.Fprintf(out, "Keys:         %s\n", strings.Join(keys, ", "))

	if len(export.Entries) > 0 {
		first := export.Entries[0]
		if verification.Partial {
			_, _ = fmt.Fprintf(out, "Start:        partial, the build predates its trail, which starts at seq 1 on %s\n", first.Time)
		} else {
			_, _ = fmt.Fprintf(out, "Start:        complete, opened by %s\n", first.Type)
		}
	}

	if verification.ChainIntact {
		_, _ = fmt.Fprintln(out, "Chain:        intact")
	} else if verification.FirstBrokenSeq > 1 {
		_, _ = fmt.Fprintf(out, "Chain:        broken at seq %d, verified up to seq %d\n", verification.FirstBrokenSeq, verification.FirstBrokenSeq-1)
	} else {
		_, _ = fmt.Fprintln(out, "Chain:        broken at seq 1")
	}

	if verification.EndorsementsValid {
		_, _ = fmt.Fprintln(out, "Endorsements: valid")
	} else {
		_, _ = fmt.Fprintf(out, "Endorsements: invalid at seq %d\n", verification.FirstInvalidEndorsementSeq)
	}

	if verification.UnendorsedFromSeq != 0 {
		_, _ = fmt.Fprintf(out, "Unendorsed:   from seq %d, written while the instance key was not provisioned\n", verification.UnendorsedFromSeq)
	}

	if len(verification.Problems) > 0 {
		_, _ = fmt.Fprintln(out, "Problems:")
		table := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		for _, problem := range verification.Problems {
			_, _ = fmt.Fprintf(table, "  seq %d\t%s\t%s\n", problem.Seq, problem.Type, problem.Message)
		}
		_ = table.Flush()
	} else {
		_, _ = fmt.Fprintln(out, "The trail is intact.")
	}
}

func init() {
	auditTrailCmd.AddCommand(auditTrailVerifyCmd)
}
