---
description: Analyze task DAG and suggest reorganization
allowed-tools:
 - Skill(jjtask)
 - Bash
 - Read
 - AskUserQuestion
model: haiku
---

<objective>
Review the current task DAG structure and suggest improvements:
- Identify dependency issues (task mentions another but isn't a child)
- Find parallelization opportunities (independent tasks that could run concurrently)
- Detect structural problems (blocked children, incomplete drafts)
</objective>

<context>
Current task DAG:
</context>

<process>
1. Run `jjtask find -s all` and `jj log -r 'tasks()` to get DAG structure
2. Log: "Reading task descriptions for dependency keywords..."
3. For each task, read description with `jjtask show-desc -r REV`
4. Log findings as you discover them:
   - "Found: mp references lv but isn't a child"
   - "Found: ky and pkm overlap - same precompact feature"
5. Present summary of all issues found
6. Propose concrete rebase commands for each issue
7. Execute rebases only with user confirmation, logging each: "Rebased X to Y"
</process>

<success_criteria>
- DAG analyzed for dependency/structure issues
- Concrete rebase commands proposed (if issues found)
- No rebases executed without user approval
</success_criteria>
