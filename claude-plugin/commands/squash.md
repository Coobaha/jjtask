---
description: Flatten @ merge into linear commit for push
argument-hint: [--keep-tasks]
allowed-tools:
 - Bash
 - AskUserQuestion
model: haiku
---

<objective>
Flatten the current @ merge into a single linear commit, ready for pushing.

Combines descriptions from all merged tasks.

Part of mega-merge workflow - see `/jjtask` for full context.
</objective>

<context>
Current @ state:
!`jj log -r @ --no-graph 2>/dev/null | head -5`

WIP tasks that will be squashed:
!`jjtask find wip 2>/dev/null || echo "no wip tasks"`
</context>

<process>
Run: `jjtask squash $ARGUMENTS`

- `jjtask squash` - flatten everything
- `jjtask squash --keep-tasks` - keep task revisions after squash
</process>
