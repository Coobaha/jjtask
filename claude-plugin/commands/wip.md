---
description: Mark task WIP and add as parent of @
argument-hint: [tasks...]
allowed-tools:
 - Bash
model: haiku
---

<objective>
Mark tasks as WIP and add them as parents of @. Multiple WIP tasks create a merge.

Part of mega-merge workflow - see `/jjtask` for full context.
</objective>

<context>
Current WIP tasks:
!`jjtask find wip 2>/dev/null || echo "no wip tasks"`

Pending tasks:
!`jjtask find todo 2>/dev/null | head -10`
</context>

<process>
Run: `jjtask wip $ARGUMENTS`

- No args: marks @ as WIP (if it's a task)
- With tasks: marks those tasks as WIP
- Multiple: `jjtask wip a b c`

Tasks are added as parents of @ (preserves @ content).
</process>
