package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"jjtask/internal/config"
	"jjtask/internal/workspace"
)

// hookEvent represents the hook event name from Claude Code
type hookEvent string

const (
	hookEventSessionStart hookEvent = "SessionStart"
	hookEventPreCompact   hookEvent = "PreCompact"
)

// hookPayload represents the JSON payload from Claude Code hooks
type hookPayload struct {
	HookEventName string `json:"hook_event_name"`
	Trigger       string `json:"trigger"` // "manual" or "auto" for PreCompact
	Source        string `json:"source"`  // "startup", "resume", "clear", "compact" for SessionStart
}

// detectHookEvent detects which Claude Code hook triggered this invocation
func detectHookEvent() (event hookEvent, trigger string) {
	// Claude Code passes payload as JSON to stdin
	// Check if stdin is a pipe (not TTY)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// TTY - no hook data
		return hookEventSessionStart, ""
	}

	// Read stdin with timeout to avoid blocking on empty pipe
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	dataCh := make(chan []byte, 1)
	go func() {
		data, _ := io.ReadAll(os.Stdin)
		dataCh <- data
	}()

	select {
	case data := <-dataCh:
		if len(data) == 0 {
			return hookEventSessionStart, ""
		}
		var p hookPayload
		if err := json.Unmarshal(data, &p); err == nil {
			switch p.HookEventName {
			case "PreCompact":
				return hookEventPreCompact, p.Trigger
			case "SessionStart":
				return hookEventSessionStart, p.Source
			}
		}
	case <-ctx.Done():
		// Timeout - no data available
	}

	return hookEventSessionStart, ""
}

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output session context for hooks",
	Long: `Output current task context for use in hooks or prompts.

This is typically used by SessionStart and PreCompact hooks to provide
context about pending tasks to AI assistants.

For PreCompact (auto), outputs task state verification instructions.
For SessionStart, outputs full quick reference.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		event, trigger := detectHookEvent()

		// PreCompact auto = context nearly full, output verification prompt
		if event == hookEventPreCompact && trigger == "auto" {
			return printPreCompactContext()
		}

		// Check for custom prime content
		customContent, hasCustom, err := config.GetPrimeContent()
		if err != nil {
			return fmt.Errorf("reading prime config: %w", err)
		}
		if hasCustom {
			fmt.Println()
			fmt.Print(customContent)
			if !strings.HasSuffix(customContent, "\n") {
				fmt.Println()
			}
			printCurrentTasks()
			return nil
		}

		fmt.Println()
		fmt.Println("## JJ TASK Quick Reference")
		fmt.Println()
		fmt.Println("Task flags: draft → todo → wip → done (also: blocked, standby, untested, review)")
		fmt.Println()

		fmt.Println("### Revsets")
		fmt.Println("tasks(), tasks_pending(), tasks_todo(), tasks_wip(), tasks_done(), tasks_blocked()")
		fmt.Println()

		fmt.Println("### Commands (all support -R, --quiet, etc.)")
		fmt.Println("jjtask create TITLE [-p REV]         Create task as child of @ (or REV)")
		fmt.Println("jjtask wip [TASKS...]                Mark WIP, add as parents of @")
		fmt.Println("jjtask done [TASKS...]               Mark done, linearize into ancestry")
		fmt.Println("jjtask drop TASKS... [--abandon]     Remove from @ (standby or abandon)")
		fmt.Println("jjtask squash                        Flatten @ merge for push")
		fmt.Println("jjtask find [-s STATUS] [-r REVSET]  List tasks (status: todo/wip/done/all)")
		fmt.Println("jjtask flag STATUS [-r REV]          Change task flag (defaults to @)")
		fmt.Println("jjtask parallel T1 T2... [-p REV]    Create sibling tasks (defaults to @)")
		fmt.Println("jjtask show-desc [-r REV]            Print revision description")
		fmt.Println("jjtask desc-transform CMD [-r REV]   Transform description with command")
		fmt.Println("jjtask checkpoint [-m MSG]           Create checkpoint commit")
		fmt.Println("jjtask stale                         Find done tasks not in @'s ancestry")
		fmt.Println("jjtask all CMD [ARGS]                Run jj CMD across workspaces")
		fmt.Println()

		fmt.Println("### Workflow")
		fmt.Println("1. `jjtask create 'task'`        # Plan tasks")
		fmt.Println("2. `jjtask wip TASK`             # Start (single=edit, multi=merge)")
		fmt.Println("3. `jj edit TASK` to work        # Work directly in task branch")
		fmt.Println("4. `jjtask done`                 # Complete, linearizes into ancestry")
		fmt.Println("5. `jjtask squash`               # Flatten for push")
		fmt.Println()
		fmt.Println("Key: @ is merge of WIP tasks. Work in task branches directly.")
		fmt.Println("For merge: `jj edit TASK`, not bare `jj absorb`.")
		fmt.Println()

		fmt.Println("### Rules")
		fmt.Println("- DAG = priority: parent tasks complete before children")
		fmt.Println("- Chain related tasks: `jjtask create --chain 'Next step'`")
		fmt.Println("- Read full spec before editing - descriptions are specifications")
		fmt.Println("- Never mark done unless ALL acceptance criteria pass")
		fmt.Println("- Use `jjtask flag review/blocked/untested` if incomplete")
		fmt.Println("- Stop and report if unsure - don't attempt JJ recovery ops")
		fmt.Println()

		fmt.Println("### Before Saying Done")
		fmt.Println("[ ] All acceptance criteria in task spec pass")
		fmt.Println("[ ] `jjtask done TASK` - mark complete")
		fmt.Println("[ ] `jjtask squash` - flatten for push when ready")
		fmt.Println()

		fmt.Println("### Native Task Tools")
		fmt.Println("TaskCreate, TaskUpdate, TaskList, TaskGet - for session workflow tracking")
		fmt.Println("Use for: multi-step work within a session, dependency ordering, progress display")
		fmt.Println("jjtask = persistent tasks in repo history; Task* = ephemeral session tracking")
		fmt.Println()

		fmt.Println("### Current Tasks")
		fmt.Println()

		repos, workspaceRoot, _ := workspace.GetRepos()

		hasTasks := false
		for _, repo := range repos {
			repoPath := workspace.ResolveRepoPath(repo, workspaceRoot)
			if printCompactTasks(repoPath, len(repos) > 1, workspace.DisplayName(repo)) {
				hasTasks = true
			}
		}
		if !hasTasks {
			fmt.Println("No tasks. Create one with: jjtask create 'Task title'")
		}

		fmt.Println()
		printCompactChanges()

		return nil
	},
}

// printCurrentTasks outputs the current tasks section
func printCurrentTasks() {
	fmt.Println()
	fmt.Println("### Current Tasks")
	fmt.Println()

	repos, workspaceRoot, _ := workspace.GetRepos()

	hasTasks := false
	for _, repo := range repos {
		repoPath := workspace.ResolveRepoPath(repo, workspaceRoot)
		if printCompactTasks(repoPath, len(repos) > 1, workspace.DisplayName(repo)) {
			hasTasks = true
		}
	}
	if !hasTasks {
		fmt.Println("No tasks. Create one with: jjtask create 'Task title'")
	}
}

// taskSection holds tasks for one status category
type taskSection struct {
	header string
	lines  []string
}

// printCompactTasks outputs tasks in compact format: id | title, aligned across all sections
// Returns true if any tasks were printed
func printCompactTasks(repoPath string, isMulti bool, repoName string) bool {
	if isMulti {
		fmt.Printf("--- %s ---\n", repoName)
	}

	// Collect all sections
	sections := []taskSection{
		{"WIP", queryTaskLines(repoPath, "tasks_wip()", "[task:wip] ")},
		{"Todo", queryTaskLines(repoPath, "tasks_todo()", "[task:todo] ")},
		{"Draft", queryTaskLines(repoPath, "tasks_draft()", "[task:draft] ")},
	}

	// Find max ID length across all sections
	maxIDLen := 0
	for _, sec := range sections {
		for _, line := range sec.lines {
			if idx := strings.Index(line, " | "); idx > maxIDLen {
				maxIDLen = idx
			}
		}
	}

	// Print each section with aligned columns
	printed := false
	first := true
	for _, sec := range sections {
		if len(sec.lines) == 0 {
			continue
		}
		if !first {
			fmt.Println()
		}
		first = false
		printed = true
		fmt.Println(sec.header + ":")
		for _, line := range sec.lines {
			if idx := strings.Index(line, " | "); idx > 0 {
				id := line[:idx]
				title := line[idx+3:] // skip " | "
				fmt.Printf("%-*s  %s\n", maxIDLen, id, title)
			}
		}
	}

	if isMulti {
		fmt.Println()
	}
	return printed
}

// queryTaskLines queries tasks and returns lines as slice
func queryTaskLines(repoPath, revset, prefix string) []string {
	tmpl := `change_id.shortest() ++ " | " ++ description.first_line().remove_prefix("` + prefix + `") ++ if(has_spec, " [desc:" ++ desc_lines ++ "L]", "") ++ "\n"`
	out, _ := client.Query("-R", repoPath, "log", "--no-graph", "-r", revset, "-T", tmpl)
	out = strings.TrimSpace(out)
	if out == "" {
		return nil
	}
	var lines []string
	for line := range strings.SplitSeq(out, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// repoChanges holds changes for one repo
type repoChanges struct {
	name   string
	files  int
	adds   int
	dels   int
	detail string
}

// printCompactChanges outputs jj diff stat in compact format across all repos
func printCompactChanges() {
	repos, workspaceRoot, _ := workspace.GetRepos()
	isMulti := len(repos) > 1

	totalFiles := 0
	totalAdds := 0
	totalDels := 0
	var repoResults []repoChanges

	for _, repo := range repos {
		repoPath := workspace.ResolveRepoPath(repo, workspaceRoot)
		out, err := client.Query("-R", repoPath, "diff", "--stat")
		if err != nil || strings.TrimSpace(out) == "" {
			continue
		}

		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) == 0 {
			continue
		}

		// Last line is summary like "5 files changed, 120 insertions(+), 45 deletions(-)"
		summary := lines[len(lines)-1]
		fileLines := lines[:len(lines)-1]

		// Parse summary
		summaryClean := strings.ReplaceAll(summary, " changed", "")
		summaryClean = strings.ReplaceAll(summaryClean, " insertions(+)", "")
		summaryClean = strings.ReplaceAll(summaryClean, " insertion(+)", "")
		summaryClean = strings.ReplaceAll(summaryClean, " deletions(-)", "")
		summaryClean = strings.ReplaceAll(summaryClean, " deletion(-)", "")
		summaryClean = strings.ReplaceAll(summaryClean, ",", "")
		parts := strings.Fields(summaryClean)

		rc := repoChanges{name: workspace.DisplayName(repo)}
		if len(parts) >= 2 {
			_, _ = fmt.Sscanf(parts[0], "%d", &rc.files)
			totalFiles += rc.files
		}
		if len(parts) >= 3 {
			_, _ = fmt.Sscanf(parts[2], "%d", &rc.adds)
			totalAdds += rc.adds
		}
		if len(parts) >= 4 {
			_, _ = fmt.Sscanf(parts[3], "%d", &rc.dels)
			totalDels += rc.dels
		}

		// Collect compact file list
		var compactFiles []string
		for _, line := range fileLines {
			fileParts := strings.Split(line, "|")
			if len(fileParts) != 2 {
				continue
			}
			filePath := strings.TrimSpace(fileParts[0])
			fileName := filePath[strings.LastIndex(filePath, "/")+1:]

			statPart := strings.TrimSpace(fileParts[1])
			statFields := strings.Fields(statPart)
			if len(statFields) == 0 {
				continue
			}

			adds := strings.Count(statPart, "+")
			dels := strings.Count(statPart, "-")

			stat := ""
			if adds > 0 {
				stat += fmt.Sprintf("+%d", adds)
			}
			if dels > 0 {
				stat += fmt.Sprintf("-%d", dels)
			}
			if stat != "" {
				compactFiles = append(compactFiles, fmt.Sprintf("%s %s", fileName, stat))
			} else {
				compactFiles = append(compactFiles, fileName)
			}
		}
		rc.detail = strings.Join(compactFiles, " | ")
		repoResults = append(repoResults, rc)
	}

	if totalFiles == 0 {
		fmt.Println("### Changes (0 files +0 -0)")
		return
	}

	// Build header with totals
	fmt.Printf("### Changes (%d files +%d -%d)\n", totalFiles, totalAdds, totalDels)

	// Output per-repo if multi-repo, otherwise just file list
	if isMulti {
		for _, rc := range repoResults {
			if rc.files == 0 {
				continue
			}
			fmt.Printf("--- %s ---\n", rc.name)
			fmt.Println(rc.detail)
		}
	} else if len(repoResults) > 0 && repoResults[0].detail != "" {
		fmt.Println(repoResults[0].detail)
	}
}

// printPreCompactContext outputs task verification when context is nearly full
func printPreCompactContext() error {
	fmt.Println()
	fmt.Println("## 🚨 Context Compacting - Verify Task State")
	fmt.Println()
	fmt.Println("Context window nearly full. Before compaction, verify:")
	fmt.Println()
	fmt.Println("### Current WIP Tasks")

	repos, workspaceRoot, _ := workspace.GetRepos()
	hasWIP := false

	for _, repo := range repos {
		repoPath := workspace.ResolveRepoPath(repo, workspaceRoot)
		out, err := client.Query("-R", repoPath, "log", "--no-graph", "-r", "tasks_wip()", "-T", "task_log_flat")
		if err == nil {
			outStr := strings.TrimRight(out, "\n")
			if outStr != "" {
				hasWIP = true
				fmt.Println(outStr)
			}
		}
	}

	if !hasWIP {
		fmt.Println("(no WIP tasks)")
	}

	fmt.Println()
	fmt.Println("### Verification Checklist")
	fmt.Println("[ ] WIP tasks still accurate? Update status if needed")
	fmt.Println("[ ] Any completed work not marked done?")
	fmt.Println("[ ] Need to create follow-up tasks before context lost?")
	fmt.Println()
	fmt.Println("### Actions")
	fmt.Println("- `jjtask find wip` - review all WIP tasks")
	fmt.Println("- `jjtask done TASK` - mark completed work")
	fmt.Println("- `jjtask create 'Follow-up'` - capture discovered work")
	fmt.Println()
	fmt.Println("Confirm with user if task state needs updates before proceeding.")
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(primeCmd)
}
