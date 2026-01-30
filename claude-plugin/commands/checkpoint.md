---
description: Create a checkpoint commit
argument-hint: [-m message]
allowed-tools:
 - Bash
model: haiku
---

<objective>
Record the current operation ID before risky operations.

Example: `jjtask checkpoint -m "Before risky rebase"`
Restore later with: `jj op restore <op-id>`
</objective>

<process>
Run: `jjtask checkpoint $ARGUMENTS`
</process>

<success_criteria>
- Operation ID displayed
- Can be restored with `jj op restore`
</success_criteria>
