package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tienkane/nodrain/internal/approval"
	"github.com/tienkane/nodrain/internal/export"
	"github.com/tienkane/nodrain/internal/scanner"
)

var (
	exportFlagSet   scanFlags
	exportOut       string
	exportUnlimited bool
)

var exportCmd = &cobra.Command{
	Use:   "export [wallet]",
	Short: "Export unsigned revoke calldata and a revoke.cash link",
	Long: `Scan a wallet and write an inert revoke bundle: a revoke.cash deep link plus
one unsigned approve(spender, 0) transaction per approval. The output is never
signed or broadcast — submit each transaction from your own wallet.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := exportFlagSet.options(args)
		if err != nil {
			return err
		}
		warnIfPaidChain(opts)

		approvals, err := scanner.Scan(cmd.Context(), opts, func(stage string) {
			fmt.Fprintln(os.Stderr, stage)
		})
		if err != nil {
			return err
		}
		if exportUnlimited {
			approvals = filterUnlimited(approvals)
		}

		bundle, err := export.Build(opts.Owner, opts.Chain.Name, opts.Chain.ID, approvals)
		if err != nil {
			return err
		}
		data, err := bundle.JSON()
		if err != nil {
			return err
		}

		if exportOut == "" {
			fmt.Println(string(data))
			return nil
		}
		if err := os.WriteFile(exportOut, data, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "wrote %d unsigned transaction(s) to %s\n", len(bundle.Transactions), exportOut)
		return nil
	},
}

func init() {
	exportFlagSet.bind(exportCmd)
	exportCmd.Flags().StringVarP(&exportOut, "out", "o", "", "write the bundle to this file instead of stdout")
	exportCmd.Flags().BoolVar(&exportUnlimited, "unlimited", false, "export only unlimited approvals")
}

func filterUnlimited(approvals []approval.Approval) []approval.Approval {
	out := approvals[:0:0]
	for _, a := range approvals {
		if a.Unlimited {
			out = append(out, a)
		}
	}
	return out
}
