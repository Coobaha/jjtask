---
description: Create a new todo task revision
argument-hint: [parent] <title> [description] [--chain] [--draft]
allowed-tools:
 - Skill(jjtask)
 - Read
 - Bash
 - AskUserQuestion
---

<objective>
Create a new jj revision marked as a todo task.

Parent defaults to @ if only title provided.

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<context>
Existing tasks:
!`jjtask find 2>/dev/null || jj log -r 'tasks_pending()' -T task_log --limit 20`

Recent commits (potential parents):
!`jj log --limit 10`
</context>

<process>
BEFORE CREATING - you MUST:
1. List any related existing tasks from context above (or state "no related tasks")
2. State which revision you'll use as parent and WHY

THEN create:

3. Run: `jjtask create $ARGUMENTS`
   - Basic: `jjtask create "title"` (parent = @)
   - With description: `jjtask create "title" "description"`
   - Custom parent: `jjtask create xyz "title"` or `jjtask create xyz "title" "description"`
   - Auto-chain: `jjtask create --chain "title"` (chains from deepest pending)
   - Draft: `jjtask create --draft "title"`
</process>

<success_criteria>
- Stated related existing tasks (or "none")
- Stated parent choice with reasoning
- New todo revision created
</success_criteria>
