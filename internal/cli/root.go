// Package cli wires the cobra command tree and translates flags into a scan.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nodrain",
	Short: "Scan a wallet for risky ERC-20 token approvals",
	Long: `nodrain is a terminal ERC-20 token-approval scanner.

It finds a wallet's active approvals across EVM chains, flags unlimited
allowances and known-risky spenders, and exports revoke.cash links plus unsigned
approve(spender, 0) calldata. nodrain never holds a private key and never signs
or broadcasts a transaction.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: `  nodrain scan 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045
  nodrain scan --chain arbitrum 0xYourWallet
  nodrain scan --json 0xYourWallet | jq '.[] | select(.unlimited)'
  nodrain export --unlimited --out bundle.json 0xYourWallet`,
}

// Execute runs the root command. It is the single entry point from main.
func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(versionCmd)

	// Enable the --version flag (alongside the `version` subcommand).
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("{{.Name}} {{.Version}} (commit " + commit + ", built " + date + ")\n")
}
