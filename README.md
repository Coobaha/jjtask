# jjtask

[![Test](https://github.com/Coobaha/jjtask/actions/workflows/ci.yml/badge.svg)](https://github.com/Coobaha/jjtask/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Structured task management using JJ (Jujutsu). Uses empty revisions as TODO markers with `[task:*]` flags, forming a DAG of plannable and executable tasks.

<img width="711" height="667" alt="jjtask log" src="https://github.com/user-attachments/assets/9784a47c-5826-4aff-b9cb-dbed4d23469a" />

## Prerequisites

- [JJ (Jujutsu)](https://martinvonz.github.io/jj/) - install with `cargo install jj-cli` or `brew install jj`

## Quick Start

```bash
# Claude Code plugin
claude plugin marketplace add Coobaha/jjtask
claude plugin install jjtask@jjtask-marketplace
```

<img width="775" height="245" alt="Claude Code installation" src="https://github.com/user-attachments/assets/81607621-6ed3-4374-92a5-c007b5f8986f" />


## Workflow

jjtask uses a "mega-merge" model: @ is always a merge of all active work.

```
┌─────────────────────────────────────────────────────────────┐
│                     PLANNING PHASE                          │
├─────────────────────────────────────────────────────────────┤
│  1. Create task DAG with specifications                     │
│     jjtask create "Add user auth" "## Requirements..."      │
│     jjtask parallel "Frontend" "Backend" "Tests"            │
│                                                             │
│  2. Review structure                                        │
│     jjtask find                                             │
│                                                             │
│  Result: Empty revisions with [task:todo] flags             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     WORKING PHASE                           │
├─────────────────────────────────────────────────────────────┤
│  3. Start working on a task (@ becomes merge of active)     │
│     jjtask wip <task-id>                                    │
│                                                             │
│  4. Work directly in @ - changes go to merged tasks         │
│     # write code, make changes                              │
│                                                             │
│  5. Complete task when ALL criteria met                     │
│     jjtask done                                             │
│                                                             │
│  6. Ready to push? Flatten the merge                        │
│     jjtask squash                                           │
└─────────────────────────────────────────────────────────────┘
```

### Workflow Rules

- Never mark `done` unless ALL acceptance criteria pass
- Use `jjtask flag blocked/review/untested` if criteria aren't fully met
- Task descriptions are specifications - follow them exactly
- @ is always a merge of all WIP + done-with-content tasks

## Task Flags

Status progression: `draft` → `todo` → `wip` → `done`

| Flag | Meaning |
| --- | --- |
| `draft` | Placeholder, needs full specification |
| `todo` | Ready to work, complete specs |
| `wip` | Work in progress |
| `blocked` | Waiting on external dependency |
| `standby` | Awaits decision |
| `untested` | Implementation done, needs testing |
| `review` | Needs review |
| `done` | Complete, all acceptance criteria met |

## Log Colors

jjtask config adds colored task flags to `jj log`:

```
○  k [task:todo] Add user authentication +12L
│     Implement OAuth2 flow with Google provider...
○  m [task:wip] Fix database connection pooling
○  n [task:done] Update dependencies
```

Colors: `todo` yellow, `wip` cyan, `done` green, `blocked` red, `draft` dim, `review` blue, `untested` magenta.

The `+12L` hint shows description length (specs with >3 lines).

### Adding to your jj log template

To show colored task flags in your custom log template, use the `task_flag` and `task_title` aliases from jjtask config:

```toml
# In your jj config [templates] section
log = '''
...
if(description.starts_with("[task:"),
  label("task " ++ task_flag, "[task:" ++ task_flag ++ "]") ++ " " ++ task_title,
  description.first_line(),
),
...
'''
```

The `label("task " ++ task_flag, ...)` applies colors defined in jjtask's `[colors]` section.

## Commands

| Command | Action |
| --- | --- |
| `jjtask create <title> [desc]` | Create task revision |
| `jjtask wip [task]` | Mark WIP, rebuild @ as merge |
| `jjtask done [task]` | Mark done (stays in @ if content) |
| `jjtask drop <task>` | Remove from @ (mark standby) |
| `jjtask squash` | Flatten @ merge for push |
| `jjtask find [-s status]` | List tasks by status |
| `jjtask flag <status> [-r rev]` | Update task status |
| `jjtask parallel <t1> <t2>...` | Create sibling tasks |
| `jjtask show-desc [-r rev]` | Print revision description |
| `jjtask checkpoint [name]` | Create named checkpoint |

Multi-repo support (requires `.jj-workspaces.yaml`):

| Command | Action |
| --- | --- |
| `jjtask all <cmd> [args]` | Run jj command across repos |

## Installation

### Option 1: PATH + Config (Recommended)

```bash
# Clone
git clone https://github.com/coobaha/jjtask.git ~/jjtask

# Add to PATH (add to ~/.bashrc or ~/.config/fish/config.fish)
export PATH="$HOME/jjtask/bin:$PATH"

# Merge config into your jj config
cat ~/jjtask/config/conf.d/10-jjtask.toml >> ~/.config/jj/config.toml
```

This gives you both `jjtask` CLI and `jj task` subcommand.

### Option 2: Fish Shell Function

```fish
# Source the function (add to config.fish for persistence)
source ~/jjtask/shell/fish/functions/jjtask-env.fish

jjtask-env       # activate (adds PATH, layers config)
jjtask-env off   # deactivate
```

### Option 3: Install Script

```bash
./install.sh              # Merge config into ~/.config/jj/config.toml
./install.sh --agent      # Agent mode: set JJ_CONFIG env var
./install.sh --uninstall  # Remove
```

## Multi-Repo Projects

Create `.jj-workspaces.yaml` in project root:

```yaml
repos:
  - path: frontend
    name: frontend
  - path: backend
    name: backend
```

Then `jjtask find` and `jjtask all` operate across all repos.

## Writing Good Task Descriptions

```
Short title (< 50 chars)

## Context
Why this task exists, what problem it solves.

## Requirements
- Specific requirement 1
- Specific requirement 2

## Acceptance criteria
- Criterion 1 (testable)
- Criterion 2 (testable)
```

## Development

Requires [mise](https://mise.jdx.dev/) for toolchain management.

```bash
mise install      # Install Go 1.25 + golangci-lint
mise run build    # Build binary to bin/jjtask-go
mise run test     # Run integration tests
mise run lint     # Run golangci-lint
```

## Documentation

- [CLAUDE.md](CLAUDE.md) - Architecture and development details
- [claude-plugin/skills/jjtask/SKILL.md](claude-plugin/skills/jjtask/SKILL.md) - Full workflow documentation

## Acknowledgments

Inspired by:
- [beads](https://github.com/steveyegge/beads) - AI-supervised issue tracker by Steve Yegge
- [jj-todo-workflow](https://github.com/YPares/agent-skills/blob/master/jj-todo-workflow/SKILL.md) - JJ-based TODO workflow skill by Yves Parès

## License

MIT
