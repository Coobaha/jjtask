---
description: Find revisions with specific task flags
argument-hint: [-s status] [-r revset]
allowed-tools:
 - Skill(jjtask)
 - Bash
 - AskUserQuestion
model: haiku
---

<objective>
List task revisions filtered by status flag or custom revset.

Without arguments: shows all pending tasks.
With -s: shows tasks with that status (pending, todo, wip, done, blocked, standby, untested, draft, review, all)
With -r: shows tasks matching custom revset

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<context>
!`jjtask find $ARGUMENTS`
</context>

<process>
1. Review the task list above
2. Suggest next actions based on task states
</process>

<success_criteria>
- Task list displayed
- Actionable suggestions provided
</success_criteria>
