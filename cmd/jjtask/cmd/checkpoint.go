package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var checkpointMessage string

var checkpointCmd = &cobra.Command{
	Use:   "checkpoint [--message MSG]",
	Short: "Record operation ID for recovery",
	Long: `Record the current jj operation ID so you can restore to this
point if something goes wrong.

Examples:
  jjtask checkpoint
  jjtask checkpoint -m "Before risky rebase"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		message := checkpointMessage

		// Get current operation ID
		opID, err := client.Query("op", "log", "--no-graph", "-T", "id.short()", "--limit", "1")
		if err != nil {
			return fmt.Errorf("failed to get operation ID: %w", err)
		}
		opID = strings.TrimSpace(opID)

		if message != "" {
			fmt.Printf("Checkpoint '%s' at operation: %s\n", message, opID)
		} else {
			fmt.Printf("Checkpoint at operation: %s\n", opID)
		}
		fmt.Printf("  Restore with: jj op restore %s\n", opID)

		// Show current state
		fmt.Println()
		fmt.Println("  Current state:")
		if err := client.Run("log", "-r", "@", "--limit", "3"); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	checkpointCmd.Flags().StringVarP(&checkpointMessage, "message", "m", "", "checkpoint message")
	rootCmd.AddCommand(checkpointCmd)
}
