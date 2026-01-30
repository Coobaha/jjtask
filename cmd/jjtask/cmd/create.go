package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	createDraft  bool
	createParent string
	createChain  bool
)

var createCmd = &cobra.Command{
	Use:   "create <title> [description]",
	Short: "Create a new task revision",
	Long: `Create a new task revision as direct child of @ (or --parent REV).

By default, creates a direct child of @. Use --chain to auto-chain from
the deepest pending descendant instead.

Examples:
  jjtask create "Fix bug"                      # direct child of @
  jjtask create --parent xyz "Fix bug"         # direct child of xyz
  jjtask create --chain "Next step"            # chain from deepest pending
  jjtask create --draft "Future work"          # draft task`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runCreate,
}

func init() {
	createCmd.Flags().BoolVar(&createDraft, "draft", false, "Create with [task:draft] flag")
	createCmd.Flags().StringVarP(&createParent, "parent", "p", "", "Create as child of REV (default: @)")
	createCmd.Flags().BoolVar(&createChain, "chain", false, "Auto-chain from deepest pending descendant")
	rootCmd.AddCommand(createCmd)
	createCmd.ValidArgsFunction = completeRevision
}

func runCreate(cmd *cobra.Command, args []string) error {
	title := args[0]
	desc := ""
	if len(args) >= 2 {
		desc = args[1]
	}

	parent := "@"
	if createParent != "" {
		parent = createParent
	}

	// Check if @ is a WIP task when using explicit parent (not @)
	if parent != "@" {
		checkWipSuggestion(cmd)
	}

	// Auto-chain: find deepest pending descendant (only with --chain flag)
	if createChain {
		leaf := findDeepestPendingDescendant(parent)
		if leaf != "" {
			parent = leaf
		}
	}

	flag := "todo"
	if createDraft {
		flag = "draft"
	}

	message := fmt.Sprintf("[task:%s] %s", flag, title)
	if desc != "" {
		message = message + "\n\n" + desc
	}

	err := client.Run("new", "--no-edit", parent, "-m", message)
	if err != nil {
		return err
	}

	// Get the created revision's change ID
	out, err := client.Query("log", "-r", "children("+parent+") & description(substring:\"[task:\") & heads(all())", "--no-graph", "-T", "change_id.shortest()", "--limit", "1")
	if err != nil {
		fmt.Printf("Created task [task:%s] %s (could not resolve ID: %v)\n", flag, title, err)
		return nil
	}
	changeID := strings.TrimSpace(out)
	if changeID == "" {
		fmt.Printf("Created task [task:%s] %s\n", flag, title)
	} else {
		fmt.Printf("Created new commit %s (empty) [task:%s] %s\n", changeID, flag, title)
	}

	return nil
}

// findDeepestPendingDescendant finds the deepest pending task descendant of rev
func findDeepestPendingDescendant(rev string) string {
	// Get all pending descendants, sorted by depth (most ancestors = deepest)
	// We want the leaf of the chain - a task with no pending children
	out, err := client.Query("log",
		"-r", fmt.Sprintf("(%s | descendants(%s)) & tasks_pending()", rev, rev),
		"--no-graph",
		"-T", `change_id.shortest() ++ "\n"`,
	)
	if err != nil {
		return ""
	}

	candidates := strings.Split(strings.TrimSpace(out), "\n")
	if len(candidates) == 0 || (len(candidates) == 1 && candidates[0] == "") {
		return ""
	}

	// Find the leaf - a candidate with no pending children
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}

		// Check if this candidate has any pending children
		children, err := client.Query("log",
			"-r", fmt.Sprintf("children(%s) & tasks_pending()", candidate),
			"--no-graph",
			"-T", "change_id.shortest()",
			"--limit", "1",
		)
		if err != nil {
			continue
		}

		if strings.TrimSpace(children) == "" {
			// No pending children - this is the leaf
			return candidate
		}
	}

	// No leaf found, return last candidate
	return strings.TrimSpace(candidates[len(candidates)-1])
}

// checkWipSuggestion suggests chaining to @ if @ is a WIP task
func checkWipSuggestion(cmd *cobra.Command) {
	atDesc, err := client.GetDescription("@")
	if err != nil {
		return
	}
	if !strings.HasPrefix(atDesc, "[task:wip]") {
		return
	}

	atID, err := client.Query("log", "-r", "@", "--no-graph", "-T", "change_id.shortest()")
	if err != nil {
		return
	}
	atID = strings.TrimSpace(atID)

	stderr := cmd.ErrOrStderr()
	_, _ = fmt.Fprintln(stderr)
	_, _ = fmt.Fprintf(stderr, "Note: Current revision (%s) is a WIP task.\n", atID)
	_, _ = fmt.Fprintln(stderr, "Consider: `jjtask create \"title\"` to auto-chain from @")
	_, _ = fmt.Fprintln(stderr)
}
