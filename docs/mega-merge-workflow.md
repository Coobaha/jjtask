# Mega-Merge Workflow

Based on [simultaneous edits](https://steveklabnik.github.io/jujutsu-tutorial/advanced/simultaneous-edits.html) and [megamerges](https://v5.chriskrycho.com/journal/jujutsu-megamerges-and-jj-absorb/).

## The Problem

With jjtask, tasks were floating on side branches:
- Create task (side branch)
- Work in @ (main line)
- Squash into task (still side branch)
- Forget to merge back → orphan branches

Result: constantly fighting with "where is my work?", manual hoist/squash operations, tasks getting lost.

## The Solution

**@ is always a merge of all WIP tasks.**

```
@ = merge(all_wip_tasks)
```

Done tasks linearize into ancestry - they become ancestors of remaining WIP tasks, creating clean linear history ready for push.

## Commands

| Command | What it does |
|---------|--------------|
| `jjtask wip [tasks...]` | Mark WIP + add as parents of @ |
| `jjtask done [tasks...]` | Mark done + linearize into ancestry |
| `jjtask drop [tasks...]` | Remove from @ merge (mark as standby) |
| `jjtask squash` | Flatten @ merge into linear commit for push |

## Workflow

```bash
# 1. Create tasks (planning)
jjtask create trunk "Feature A"
jjtask create trunk "Feature B"

# 2. Start work - @ becomes merge of WIP tasks
jjtask wip feature-a
jjtask wip feature-b

# 3. Work in task branches directly, or use merge for visibility
jj edit feature-a   # Work directly in the task
# or stay in merge and use: jj absorb --into 'tasks_wip()'

# 4. Complete a task - drops out of merge
jjtask done feature-a

# 5. Ready to push? Flatten the merge
jjtask squash
```

## How It Works

**wip**: Adds task as parent of @ using `jj rebase -r @ -o existing_parents -o new_task`. Preserves @ content.

**done**: Marks task done, then linearizes - rebases other parents onto the done task, so it becomes an ancestor instead of a floating branch.

**drop**: Removes task from @ parents using rebase. Marks as standby.

All operations preserve @ content through rebase rather than creating new commits.

## Routing Changes to Tasks

When working in a merge, you need to get changes into the right task branch.

**Option 1: Work directly in task branch (safest)**
```bash
jj edit task-a        # Work directly in the task
# make changes...
jj new task-a task-b  # Recreate merge to see combined state
```

**Option 2: Use absorb with explicit targets**
```bash
jj absorb --into 'tasks_wip()'  # Only absorb into WIP tasks, not ancestors
```

**Avoid bare `jj absorb`** - it routes changes based on where lines were last modified, which could send edits to ancestor commits deep in history if you're modifying lines that weren't touched by your task branches.

## Benefits

- **No orphan branches** - work is always in @
- **See conflicts immediately** - merge commit shows them
- **Clean separation** - done work drops out, WIP stays
- **Linear history when ready** - `squash` flattens for push

## Edge Cases

- **Conflicts between WIP tasks**: @ shows conflict markers immediately
- **Want to isolate one task**: `jj edit task` still works
- **Stale branches**: Done tasks not in @'s ancestry should be abandoned

## Parallel Work (Multiple Agents)

For multiple Claude sessions working simultaneously, use jj workspaces:

```bash
# Terminal 1: Create workspace for task A
jj workspace add .workspaces/agent-a --revision task-a
cd .workspaces/agent-a
# work...
jjtask done

# Terminal 2: Create workspace for task B
jj workspace add .workspaces/agent-b --revision task-b
cd .workspaces/agent-b
# work...
jjtask done

# Main workspace sees merged state automatically
```

Each workspace has its own @ that mega-merge rebuilds independently.

## Implementation

Core helpers in `internal/jj/jj.go`:
- `GetParents(rev)` - returns parent change IDs
- `AddToMerge(task)` - adds task as parent of @ via rebase
- `RemoveFromMerge(task)` - removes task from @ parents via rebase
