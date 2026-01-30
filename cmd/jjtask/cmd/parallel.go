package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	parallelDraft  bool
	parallelParent string
)

var parallelCmd = &cobra.Command{
	Use:   "parallel <title1> <title2> [title3...] [--parent REV]",
	Short: "Create sibling tasks under parent",
	Long: `Create multiple parallel task branches from the same parent.

Examples:
  jjtask parallel "Widget A" "Widget B" "Widget C"
  jjtask parallel --draft --parent mxyz "Future A" "Future B"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		parent := parallelParent
		titles := args

		flag := "todo"
		if parallelDraft {
			flag = "draft"
		}

		for _, title := range titles {
			message := fmt.Sprintf("[task:%s] %s", flag, title)
			if err := client.Run("new", "--no-edit", parent, "-m", message); err != nil {
				return fmt.Errorf("failed to create task %q: %w", title, err)
			}
		}

		fmt.Printf("Created %d parallel task branches from %s\n", len(titles), parent)
		return nil
	},
}

func init() {
	parallelCmd.Flags().BoolVar(&parallelDraft, "draft", false, "Create with [task:draft] flag")
	parallelCmd.Flags().StringVarP(&parallelParent, "parent", "p", "@", "parent revision for all tasks")
	rootCmd.AddCommand(parallelCmd)
	_ = parallelCmd.RegisterFlagCompletionFunc("parent", completeRevision)
}
