package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done [tasks...]",
	Short: "Mark tasks done and linearize into ancestry",
	Long: `Mark tasks as done. When task is a parent of @, linearizes it into the ancestry.

If the task is a merge parent, other parents are rebased onto the done task,
making it part of the linear history rather than a floating branch.

Examples:
  jjtask done xyz       # Mark xyz as done
  jjtask done           # Mark @ as done (if it's a task)
  jjtask done a b c     # Mark multiple tasks done`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		revs := args
		if len(revs) == 0 {
			revs = []string{"@"}
		}

		var orphans []string
		for _, rev := range revs {
			isOrphan, err := markDone(cmd, rev)
			if err != nil {
				return fmt.Errorf("failed to mark %s done: %w", rev, err)
			}
			if isOrphan {
				changeID, _ := client.Query("log", "-r", rev, "--no-graph", "-T", "change_id.shortest()")
				orphans = append(orphans, strings.TrimSpace(changeID))
			}
		}

		if len(orphans) > 0 {
			printOrphanWarning(orphans)
		}

		return nil
	},
}

// markDone marks a task as done and linearizes if it's a merge parent.
// Returns true if the task is an orphan (not in @'s ancestry after marking done).
func markDone(cmd *cobra.Command, rev string) (isOrphan bool, err error) {
	// Get change ID
	changeID, err := client.Query("log", "-r", rev, "--no-graph", "-T", "change_id.shortest()")
	if err != nil {
		return false, fmt.Errorf("getting change ID: %w", err)
	}
	changeID = strings.TrimSpace(changeID)

	// Check if task is a parent of @
	parents, err := client.GetParents("@")
	if err != nil {
		return false, fmt.Errorf("getting @ parents: %w", err)
	}

	isParent := false
	var otherParents []string
	for _, p := range parents {
		if p == changeID {
			isParent = true
		} else {
			otherParents = append(otherParents, p)
		}
	}

	// Warn about empty task or uncommitted work before marking done
	checkEmptyTask(cmd, changeID)
	checkWorkingCopyDiff(cmd, changeID, "done")

	// Mark as done
	if err := setTaskFlag(rev, "done"); err != nil {
		return false, fmt.Errorf("setting flag: %w", err)
	}

	// If task is a merge parent, linearize: rebase other parents onto done task
	if isParent && len(otherParents) > 0 {
		if err := linearizeDoneTask(changeID, otherParents); err != nil {
			return false, fmt.Errorf("linearizing: %w", err)
		}
	}

	// Check if task ended up in @'s ancestry
	inAncestry, err := client.IsAncestorOf(changeID, "@")
	if err != nil {
		return false, nil // Ignore error, just skip orphan check
	}

	return !inAncestry, nil
}

// linearizeDoneTask rebases other merge parents onto the done task,
// then rebases @ onto the top of the new linear chain
func linearizeDoneTask(doneTask string, otherParents []string) error {
	// Rebase each other parent (with descendants) onto the done task
	base := doneTask
	for _, parent := range otherParents {
		if err := client.Run("rebase", "-s", parent, "-o", base); err != nil {
			return fmt.Errorf("rebasing %s onto %s: %w", parent, base, err)
		}
		base = parent
	}

	// Rebase @ onto the last parent (now linear on top of done task)
	if err := client.Run("rebase", "-s", "@", "-o", base); err != nil {
		return fmt.Errorf("rebasing @ onto %s: %w", base, err)
	}

	return nil
}

func printOrphanWarning(orphans []string) {
	revList := strings.Join(orphans, " ")
	revUnion := strings.Join(orphans, " | ")

	_, _ = fmt.Fprintf(os.Stderr, "\nWarning: %s marked done but not in @'s ancestry (orphan tasks)\n", revList)
	_, _ = fmt.Fprintln(os.Stderr, "These tasks were never 'wip' - their specs won't be in linear history.")
	_, _ = fmt.Fprintln(os.Stderr)
	_, _ = fmt.Fprintln(os.Stderr, "Options:")
	_, _ = fmt.Fprintln(os.Stderr, "  1. Consolidate specs into @ description, then abandon tasks")
	_, _ = fmt.Fprintln(os.Stderr, "  2. Linearize into ancestry (may conflict)")
	_, _ = fmt.Fprintln(os.Stderr, "  3. Leave as-is (manual cleanup later)")
	_, _ = fmt.Fprintln(os.Stderr)
	_, _ = fmt.Fprintf(os.Stderr, "View specs: jj log -r '%s' --no-graph -T description\n", revUnion)
	_, _ = fmt.Fprintln(os.Stderr)
}

func init() {
	rootCmd.AddCommand(doneCmd)
}
