---
description: Update task flag on a revision
argument-hint: <flag> [-r rev]
allowed-tools:
 - Skill(jjtask)
 - Bash
 - AskUserQuestion
model: haiku
---

<objective>
Change the task status flag on a revision.

For common transitions, use dedicated commands:
- `jjtask wip TASK` - mark WIP and rebuild @ as merge
- `jjtask done TASK` - mark done (stays in @ if has content)
- `jjtask drop TASK` - remove from @ without completing

For other flags (draft, blocked, standby, untested, review), use this command.

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<context>
Current tasks:
!`jjtask find 2>/dev/null | head -20`
</context>

<process>
1. Run: `jjtask flag $ARGUMENTS`
2. Confirm the flag was updated
</process>

<success_criteria>
- Task flag updated
- No conflicts created
</success_criteria>
