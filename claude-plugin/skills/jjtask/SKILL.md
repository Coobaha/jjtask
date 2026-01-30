---
name: jjtask
description: Structured TODO commit workflow using JJ (Jujutsu). Use to plan tasks as empty commits with [task:*] flags, track progress through status transitions, manage parallel task DAGs with dependency checking. Enforces completion discipline. Enables to divide work between Planners and Workers.
---

<context>
Designed for JJ 0.36.x+. Uses revset aliases and templates defined in jjtask config.
</context>

<objective>
Manage a DAG of empty revisions as TODO markers representing tasks. Revision descriptions act as specifications. Two roles: Planners (create empty revisions with specs) and Workers (implement them). For JJ basics (revsets, commands, recovery), see the `/jj` skill.
</objective>

<quick_start>

```bash
# 1. Plan: Create TODO tasks
jjtask create "Add user validation" "Check email format and password strength"
jjtask create --chain "Add validation tests" "Test valid/invalid emails and passwords"

# 2. Start working on a task
jjtask wip abc123
# Single task: @ becomes the task (jj edit)
# Multiple WIP: @ becomes merge commit

# 3. Work and complete
# For single task: work directly in @
# For merge: jj edit TASK to work in specific task
jjtask done abc123
# Task linearizes into ancestry (becomes ancestor of remaining WIP)

# 4. Flatten for push
jjtask squash
# Squashes all merged task content into linear commit
```
</quick_start>

<success_criteria>
- Task created with correct parent relationship
- Status flags reflect actual task state
- DAG shows clear priority (chained tasks) and parallelism (sibling tasks)
- All acceptance criteria met before marking done
- @ is always a merge of all WIP tasks
</success_criteria>

<status_flags>

| Flag              | Meaning                                      |
| ----------------- | -------------------------------------------- |
| `[task:draft]`    | Placeholder, needs full specification        |
| `[task:todo]`     | Ready to work, complete specs                |
| `[task:wip]`      | Work in progress                             |
| `[task:blocked]`  | Waiting on external dependency               |
| `[task:standby]`  | Awaits decision                              |
| `[task:untested]` | Implementation done, needs testing           |
| `[task:review]`   | Needs review                                 |
| `[task:done]`     | Complete, all acceptance criteria met        |

Progression: `draft` -> `todo` -> `wip` -> `done`

```bash
jjtask wip xyz        # Mark xyz as WIP, add as parent of @
jjtask wip a b c      # Mark multiple as WIP
jjtask done xyz       # Mark done, linearizes into ancestry
jjtask done a b c     # Mark multiple as done
jjtask drop xyz       # Remove from @ without completing
jjtask flag review    # Other flags via generic command
```
</status_flags>

<description_management>

Flag changes only update status. To modify description content:

```bash
# Add completion notes when marking done
jjtask done xyz
jj desc -r xyz -m "$(jjtask show-desc -r xyz)

## Completion
- What was done
- Deviations from spec"

# Check off acceptance criteria
jjtask desc-transform 's/- \[ \] First criterion/- [x] First criterion/'

# Append a section
jjtask desc-transform 's/$/\n\n## Notes\nAdditional context here/'

# Batch update multiple tasks
jjtask batch-desc 's/old-term/new-term/g' -r 'tasks_todo()'
```

When to use what:
- `jjtask flag` - status only
- `jj desc -r REV -m "..."` - replace entire description
- `jjtask desc-transform` - partial find/replace with sed
- `jjtask batch-desc` - same transform across multiple tasks
</description_management>

<finding_tasks>

```bash
jjtask find             # Pending tasks with DAG structure
jjtask find -s todo     # Only [task:todo]
jjtask find -s wip      # Only [task:wip]
jjtask find -s done     # Only [task:done]
jjtask find -s all      # All tasks including done
```
</finding_tasks>

<parallel_tasks>

```bash
# Create parallel branches from @ (default parent)
jjtask parallel "Widget A" "Widget B" "Widget C"

# Or specify parent explicitly
jjtask parallel --parent xyz123 "Widget A" "Widget B"

# Merge point (all parents must complete)
jj new --no-edit <A-id> <B-id> <C-id> -m "[task:todo] Integration\n\n..."
```
</parallel_tasks>

<todo_description_format>

```
Short title (< 50 chars)

## Context
Why this task exists, what problem it solves.

## Requirements
- Specific requirement 1
- Specific requirement 2

## Acceptance criteria
- Criterion 1 (testable)
- Criterion 2 (testable)
```
</todo_description_format>

<completion_discipline>

Do NOT mark done unless ALL acceptance criteria are met.

Mark done when:
- Every requirement implemented
- All acceptance criteria pass
- Tests pass

Never mark done when:
- "Good enough" or "mostly works"
- Tests failing
- Partial implementation
</completion_discipline>

<anti_patterns>
<pitfall name="stop-and-report">
If you encounter these issues, STOP and report:
- Made changes in wrong revision
- Previous work needs fixes
- Uncertain about how to proceed
- Dependencies unclear

Do NOT attempt to fix using JJ operations not in this workflow.
</pitfall>
</anti_patterns>

<dag_validation>

When reviewing tasks with `jjtask find`, look for structural issues:

Good DAG - chained tasks show priority, parallel tasks are siblings:
```
o  E [todo] Feature complete   <- gate: all children done, tests pass, reviewed
|-+-,
| | o  D2 [todo] Write docs    <- parallel with D1
| o |  D1 [todo] Add tests     <- parallel with D2
|-' |
o   |  C [todo] Implement      <- after B
o  -'  B [todo] Design API     <- after A
o  A [todo] Research           <- do first
@  current work
```
Reading bottom-up: A -> B -> C -> (D1 || D2) -> E (gate)

Task E is a "gate" - marks feature complete only when all children done.

Bad DAG - all siblings, no priority visible:
```
| o  E [todo] Deploy
|-'
| o  D [todo] Write docs
|-'
| o  C [todo] Implement
|-'
| o  B [todo] Design API
|-'
| o  A [todo] Research
|-'
@  current work
```
Problem: Which task comes first? No way to tell.
Fix: Chain dependent tasks with `jj rebase -s B -o A`

Dependency problems:
- Task mentions another task but isn't a child of it -> `jj rebase -s TASK -o DEPENDENCY`
- Task requires output from another but they're siblings -> rebase to make sequential
- Keywords: "after", "requires", "depends on", "once X is done", "needs"

Parallelization opportunities:
- Sequential tasks that don't share state -> could be parallel siblings
- Independent features under same parent -> good candidates for parallel agents

Structural issues:
- Done tasks with pending children -> children may be blocked
- Draft tasks mixed with todo -> drafts need specs before work begins
</dag_validation>

<working_in_merge>

When @ is a merge of multiple WIP tasks:

**Recommended: Work directly in task branch**
```bash
jj edit task-a        # Switch to working in the task
# make changes...
jjtask wip task-a     # Rebuild merge to see combined state
```

**Alternative: Use absorb with explicit targets**
```bash
jj absorb --into 'tasks_wip()'  # Only route to WIP tasks
```

**Avoid bare `jj absorb`** - it may route changes to ancestor commits if you're editing lines not touched by your task branches.
</working_in_merge>

<squashing>

After tasks are complete, flatten the merge for a clean push:

```bash
jjtask squash
# Combines all merged task content into a single linear commit
# Task descriptions become bullet points in commit message
```
</squashing>

<commands>

| Command                                  | Purpose                            |
| ---------------------------------------- | ---------------------------------- |
| `jjtask create TITLE [-p REV] [--chain]` | Create TODO (direct child of @)    |
| `jjtask wip [TASKS...]`                  | Mark WIP, add as parents of @      |
| `jjtask done [TASKS...]`                 | Mark done, linearize into ancestry |
| `jjtask drop TASKS... [--abandon]`       | Remove from @ (standby or abandon) |
| `jjtask squash`                          | Flatten @ merge for push           |
| `jjtask parallel T1 T2... [-p REV]`      | Create parallel TODOs              |
| `jjtask flag STATUS [-r REV]`            | Update status flag (defaults to @) |
| `jjtask find [-s STATUS] [-r REVSET]`    | Find tasks by status or revset     |
| `jjtask show-desc [-r REV]`              | Print revision description         |
| `jjtask desc-transform CMD [-r REV]`     | Transform description with command |
| `jjtask batch-desc EXPR -r REVSET`       | Transform multiple descriptions    |
| `jjtask checkpoint [-m MSG]`             | Create named checkpoint            |
| `jjtask stale`                           | Find done tasks not in @'s ancestry|
| `jjtask all <cmd> [args]`                | Run jj command across all repos    |
| `jjtask prime`                           | Output session context for hooks   |
</commands>

<multi_repo>

Create `.jj-workspaces.yaml` in project root:

```yaml
repos:
  - path: frontend
    name: frontend
  - path: backend
    name: backend
```

Scripts show output grouped by repo. Use `jjtask all log` or `jjtask all diff` across repos.
</multi_repo>

<parallel_agents>

Multiple Claude agents can work simultaneously using jj workspaces:

```bash
# Terminal 1: Agent working on task A
jj workspace add .workspaces/agent-a --revision task-a
cd .workspaces/agent-a
# work...
jjtask done  # Rebuilds this workspace's @

# Terminal 2: Agent working on task B
jj workspace add .workspaces/agent-b --revision task-b
cd .workspaces/agent-b
# work...
jjtask done  # Rebuilds this workspace's @

# Cleanup when done
jj workspace forget agent-a
rm -rf .workspaces/agent-a
```

Each workspace has its own @ that mega-merge rebuilds independently.
No special coordination needed - jj handles workspace isolation.
</parallel_agents>

<references>
- `references/batch-operations.md` - Batch description transformations
- `references/command-syntax.md` - JJ command flag details
</references>
