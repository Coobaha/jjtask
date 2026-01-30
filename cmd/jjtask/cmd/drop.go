package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var dropAbandon bool

var dropCmd = &cobra.Command{
	Use:   "drop <tasks...>",
	Short: "Remove tasks from @ merge",
	Long: `Remove tasks from the @ merge without marking them done.

By default, marks tasks as 'standby' so they can be re-added later.
Use --abandon to permanently remove the tasks.

Examples:
  jjtask drop xyz            # Mark as standby, remove from @
  jjtask drop a b c          # Drop multiple tasks
  jjtask drop --abandon xyz  # Abandon task entirely`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, rev := range args {
			if err := dropTask(rev); err != nil {
				return fmt.Errorf("failed to drop %s: %w", rev, err)
			}
		}
		return nil
	},
}

func dropTask(rev string) error {
	// Get change ID
	changeID, err := client.Query("log", "-r", rev, "--no-graph", "-T", "change_id.shortest()")
	if err != nil {
		return fmt.Errorf("getting change ID: %w", err)
	}
	changeID = strings.TrimSpace(changeID)

	if dropAbandon {
		if err := client.Run("abandon", rev); err != nil {
			return fmt.Errorf("abandoning: %w", err)
		}
		fmt.Printf("Abandoned %s\n", rev)
	} else {
		if err := setTaskFlag(rev, "standby"); err != nil {
			return fmt.Errorf("setting flag: %w", err)
		}
		fmt.Printf("Marked %s as standby\n", rev)
	}

	// Remove from @ merge (preserves @ content)
	if err := client.RemoveFromMerge(changeID); err != nil {
		return fmt.Errorf("removing from merge: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(dropCmd)
	dropCmd.Flags().BoolVar(&dropAbandon, "abandon", false, "Abandon the tasks entirely")
}
