---
description: Create a checkpoint commit
argument-hint: [name]
allowed-tools:
 - Bash
---

<objective>
Record the current operation ID before risky operations.

Restore later with: `jj op restore <op-id>`
</objective>

<process>
Run: `jjtask checkpoint $ARGUMENTS`
</process>
