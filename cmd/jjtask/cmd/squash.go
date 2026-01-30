package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var squashKeepTasks bool

var squashCmd = &cobra.Command{
	Use:   "squash",
	Short: "Flatten @ merge into linear commit",
	Long: `Flatten the current @ merge into a single linear commit.

This takes all the merged task commits and squashes them into one commit,
ready for pushing. The commit message combines descriptions from all tasks.

Examples:
  jjtask squash              # Flatten everything
  jjtask squash --keep-tasks # Keep task revisions after squash`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get parents of @ (the merged tasks)
		parentsOut, err := client.Query("log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
		if err != nil {
			return fmt.Errorf("failed to get parents: %w", err)
		}

		var parents []string
		for _, line := range strings.Split(strings.TrimSpace(parentsOut), "\n") {
			if line != "" {
				parents = append(parents, line)
			}
		}

		if len(parents) == 0 {
			fmt.Println("No parents to squash")
			return nil
		}

		if len(parents) == 1 {
			fmt.Println("Only one parent, nothing to merge-squash")
			return nil
		}

		// Build combined commit message from task descriptions
		var msgParts []string
		for _, p := range parents {
			desc, err := client.GetDescription(p)
			if err != nil || desc == "" {
				continue
			}
			// Strip [task:*] prefix for cleaner message
			desc = strings.TrimSpace(desc)
			if strings.HasPrefix(desc, "[task:") {
				if idx := strings.Index(desc, "]"); idx != -1 {
					desc = strings.TrimSpace(desc[idx+1:])
				}
			}
			if desc != "" {
				msgParts = append(msgParts, "- "+strings.Split(desc, "\n")[0])
			}
		}

		combinedMsg := "Squashed tasks:\n" + strings.Join(msgParts, "\n")

		// Squash all parents into @
		if err := client.Run("squash", "--from", "parents(@)", "--message", combinedMsg); err != nil {
			return fmt.Errorf("failed to squash: %w", err)
		}

		fmt.Printf("Squashed %d tasks into linear commit\n", len(parents))

		if !squashKeepTasks {
			// Mark original tasks as done (they're now empty after squash)
			for _, p := range parents {
				desc, err := client.GetDescription(p)
				if err != nil {
					continue
				}
				if strings.Contains(desc, "[task:wip]") {
					newDesc := strings.Replace(desc, "[task:wip]", "[task:done]", 1)
					_ = client.SetDescription(p, newDesc)
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(squashCmd)
	squashCmd.Flags().BoolVar(&squashKeepTasks, "keep-tasks", false, "Keep task revisions after squash")
}
