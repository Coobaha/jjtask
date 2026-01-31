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

// linearizeDoneTask integrates the done task into @'s linear ancestry.
// It identifies work branches (non-task parents) and task branches,
// then rebases tasks ON TOP of work (tasks are newer, work is older).
//
// Strategy:
// 1. Find the work tip (newest work commit among parents)
// 2. Rebase done task onto work tip
// 3. Chain other task parents onto done task
// 4. Rebase @ onto the task chain tip
//
// Result: work1 → work2 → work3 → taskA → taskB → @
func linearizeDoneTask(doneTask string, otherParents []string) error {
	// Separate parents into tasks and work commits
	var taskParents, workParents []string
	for _, parent := range otherParents {
		if isTaskCommit(parent) {
			taskParents = append(taskParents, parent)
		} else {
			workParents = append(workParents, parent)
		}
	}

	// If no work parents, use old behavior: chain everything onto done task
	if len(workParents) == 0 {
		base := doneTask
		for _, parent := range otherParents {
			if err := client.Run("rebase", "-s", parent, "-o", base); err != nil {
				return fmt.Errorf("rebasing %s onto %s: %w", parent, base, err)
			}
			base = parent
		}
		if err := client.Run("rebase", "-s", "@", "-o", base); err != nil {
			return fmt.Errorf("rebasing @ onto %s: %w", base, err)
		}
		return nil
	}

	// Find the work tip (newest among work parents)
	workTip, err := findWorkTip(workParents)
	if err != nil {
		return fmt.Errorf("finding work tip: %w", err)
	}

	// Rebase done task onto work tip (tasks go ON TOP of work)
	if err := client.Run("rebase", "-s", doneTask, "-o", workTip); err != nil {
		return fmt.Errorf("rebasing done task %s onto %s: %w", doneTask, workTip, err)
	}

	// Chain other task parents onto done task
	taskTip := doneTask
	for _, taskParent := range taskParents {
		if err := client.Run("rebase", "-s", taskParent, "-o", taskTip); err != nil {
			return fmt.Errorf("rebasing task %s onto %s: %w", taskParent, taskTip, err)
		}
		taskTip = taskParent
	}

	// Rebase @ onto the task chain tip
	if err := client.Run("rebase", "-s", "@", "-o", taskTip); err != nil {
		return fmt.Errorf("rebasing @ onto %s: %w", taskTip, err)
	}

	return nil
}

// isTaskCommit checks if a revision is a task commit (has [task:*] in description)
func isTaskCommit(rev string) bool {
	desc, err := client.Query("log", "-r", rev, "--no-graph", "-T", "description")
	if err != nil {
		return false
	}
	return strings.Contains(desc, "[task:")
}

// findWorkTip finds the newest commit among work parents
func findWorkTip(workParents []string) (string, error) {
	if len(workParents) == 1 {
		return workParents[0], nil
	}

	// Find the tip (head) of work branches
	revset := strings.Join(workParents, " | ")
	out, err := client.Query("log", "-r", "heads("+revset+")", "--no-graph", "-T", "change_id.shortest()", "--limit", "1")
	if err != nil {
		return workParents[0], nil
	}
	tip := strings.TrimSpace(out)
	if tip == "" {
		return workParents[0], nil
	}
	return tip, nil
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
