package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var staleCmd = &cobra.Command{
	Use:   "stale",
	Short: "Find done tasks not in current line of work",
	Long: `Find done tasks that are not ancestors of @.

These are "stale" branches - work that was completed on a side branch
but may not have been integrated into your current work. They might be:
- Superseded (work landed via different commits)
- Orphaned (forgot to merge)
- Exploratory (intentionally abandoned)

Use jj abandon to clean up superseded/orphaned tasks.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Find done tasks not in @'s ancestry
		out, err := client.Query("log", "-r", "tasks_done() ~ ::@", "--no-graph", "-T", `change_id.shortest() ++ " " ++ description.first_line() ++ "\n"`)
		if err != nil {
			return fmt.Errorf("failed to find stale tasks: %w", err)
		}

		out = strings.TrimSpace(out)
		if out == "" {
			fmt.Println("No stale done tasks found")
			return nil
		}

		fmt.Println("Stale done tasks (not in @'s ancestry):")
		fmt.Println()
		for _, line := range strings.Split(out, "\n") {
			if line != "" {
				fmt.Println("  " + line)
			}
		}
		fmt.Println()
		fmt.Println("These may be superseded, orphaned, or exploratory.")
		fmt.Println("Use `jj abandon REV` to clean up, or `jj rebase -s REV -d @` to integrate.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(staleCmd)
}
