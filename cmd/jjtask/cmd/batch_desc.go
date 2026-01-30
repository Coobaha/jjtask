package cmd

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var batchDescRevset string

var batchDescCmd = &cobra.Command{
	Use:   "batch-desc <sed-expr> --revset REVSET",
	Short: "Transform multiple revision descriptions",
	Long: `Apply a sed transformation to all revisions matching a revset.

Examples:
  jjtask batch-desc 's/old/new/' --revset 'tasks_todo()'
  jjtask batch-desc 's/WIP/DONE/' -r 'description(substring:"WIP")'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sedExpr := args[0]
		revset := batchDescRevset

		if revset == "" {
			return fmt.Errorf("--revset is required")
		}

		// Get matching revisions
		out, err := client.Query("log", "-r", revset, "--no-graph", "-T", "change_id.shortest() ++ \"\\n\"")
		if err != nil {
			return fmt.Errorf("failed to query revisions: %w", err)
		}

		revs := strings.Split(strings.TrimSpace(out), "\n")
		if len(revs) == 0 || (len(revs) == 1 && revs[0] == "") {
			fmt.Println("No matching revisions")
			return nil
		}

		changed := 0
		for _, rev := range revs {
			rev = strings.TrimSpace(rev)
			if rev == "" {
				continue
			}

			desc, err := client.GetDescription(rev)
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to get description for %s: %v\n", rev, err)
				continue
			}

			// Run sed
			sedCmd := exec.Command("sed", sedExpr)
			sedCmd.Stdin = strings.NewReader(desc)
			var stdout, stderr bytes.Buffer
			sedCmd.Stdout = &stdout
			sedCmd.Stderr = &stderr

			if err := sedCmd.Run(); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: sed failed for %s: %v\n", rev, err)
				continue
			}

			newDesc := stdout.String()
			if newDesc == desc {
				continue
			}

			if err := client.SetDescription(rev, newDesc); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to update %s: %v\n", rev, err)
				continue
			}

			changed++
			fmt.Printf("Updated %s\n", rev)
		}

		fmt.Printf("Modified %d of %d revisions\n", changed, len(revs))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(batchDescCmd)
	batchDescCmd.Flags().StringVarP(&batchDescRevset, "revset", "r", "", "revset to match revisions (required)")
	_ = batchDescCmd.MarkFlagRequired("revset")
	_ = batchDescCmd.RegisterFlagCompletionFunc("revset", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"tasks()",
			"tasks_pending()",
			"tasks_todo()",
			"tasks_wip()",
			"tasks_done()",
		}, cobra.ShellCompDirectiveNoFileComp
	})
}
