---
description: Mark task done and linearize into ancestry
argument-hint: [tasks...]
allowed-tools:
 - Bash
 - AskUserQuestion
model: haiku
---

<objective>
Mark tasks as done. Done tasks linearize into ancestry - other WIP parents rebase onto them, creating linear history.

Part of mega-merge workflow - see `/jjtask` for full context.
</objective>

<context>
Current WIP tasks:
!`jjtask find wip 2>/dev/null || echo "no wip tasks"`
</context>

<process>
Run: `jjtask done $ARGUMENTS`

- No args: marks current task (@) as done
- With tasks: marks those tasks as done
- Multiple: `jjtask done a b c`

Done tasks become ancestors of remaining WIP tasks.
</process>
