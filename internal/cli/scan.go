package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tienkane/nodrain/internal/approval"
	"github.com/tienkane/nodrain/internal/scanner"
	"github.com/tienkane/nodrain/internal/tui"
)

var (
	scanFlagSet scanFlags
	scanJSON    bool
)

var scanCmd = &cobra.Command{
	Use:   "scan [wallet]",
	Short: "Scan a wallet for active token approvals",
	Long: `Scan a wallet for active ERC-20 approvals.

By default an interactive table opens: navigate with ↑/↓, tick approvals with
space, and press enter to review and export an unsigned revoke bundle. Use
--json for scriptable output.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := scanFlagSet.options(args)
		if err != nil {
			return err
		}
		warnIfPaidChain(opts)
		if scanJSON {
			return runHeadlessScan(cmd.Context(), opts)
		}
		return tui.Run(cmd.Context(), opts)
	},
}

func init() {
	scanFlagSet.bind(scanCmd)
	scanCmd.Flags().BoolVar(&scanJSON, "json", false, "print approvals as JSON instead of opening the interactive table")
}

// runHeadlessScan performs a scan and writes the approvals as JSON to stdout.
// Progress goes to stderr so stdout stays a clean, pipeable JSON document.
func runHeadlessScan(ctx context.Context, opts scanner.Options) error {
	approvals, err := scanner.Scan(ctx, opts, func(stage string) {
		fmt.Fprintln(os.Stderr, stage)
	})
	if err != nil {
		return err
	}
	if approvals == nil {
		approvals = []approval.Approval{}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(approvals)
}
