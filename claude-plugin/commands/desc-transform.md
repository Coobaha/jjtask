---
description: Transform a revision description with sed
argument-hint: <sed-expr> [-r rev]
allowed-tools:
 - Bash
 - AskUserQuestion
model: haiku
---

<objective>
Pipe a revision's description through sed and update it.

Example: `jjtask desc-transform 's/foo/bar/'` (defaults to @)
Example: `jjtask desc-transform 's/foo/bar/' -r xyz`
</objective>

<process>
Run: `jjtask desc-transform $ARGUMENTS`
</process>

<success_criteria>
- Description transformed successfully
- No errors from sed expression
</success_criteria>
