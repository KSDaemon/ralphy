# ralphy

An autonomous AI coding agent loop with a terminal admin dashboard.

![Ralphy Attraction picture](misc/ralphy-in-action.webp)

Originally based on [snarktank/ralph](https://github.com/snarktank/ralph) — extended with git worktree support, session management, 
and a Go-based TUI for monitoring.

**Important:** The core value lives in the prompts — [prompt.md](./prompt.md) and [skills/](./skills/) — which define how the agent plans, implements, verifies, 
and commits code autonomously, one user story per iteration.

## What Is This

The implementation has two parts:

1. **`bin/ralphy`** — a self-contained shell script that drives the agent loop. Run `ralphy help` to see usage. Supports both **branching mode** 
(work in the current repo) and **git worktree mode** (default — creates an isolated worktree for the feature branch). Works with OpenCode, 
Claude Code, and Amp as the underlying AI tool.

2. **`bin/ralphy-admin`** — a terminal UI for monitoring running sessions, viewing logs and progress in real time, 
and managing sessions (pause, resume, kill). Vim-style navigation.

![Ralphy Admin Demo](misc/ralphy-admin-demo.gif)

## Quick Start

```bash
# Clone
git clone https://github.com/ksdaemon/ralphy.git
cd ralphy

# Install skills into your AI tool config directories (ralphy will autodetect tools)
bin/ralphy install

# Add bin/ to your PATH for convenience
export PATH="$PWD/bin:$PATH"

# Navigate to your project directory and start a session
cd ~/your-project
ralphy   # starts the agent loop
```

`ralphy install` symlinks the skills (prd, ralph) into `~/.config/opencode/skills/` and `~/.claude/skills/` for whichever tools are detected in your PATH.

## Building the Admin TUI

```bash
cd admin
make build    # produces bin/ralphy-admin
```

Requires Go 1.25+. The binary is placed into `bin/` — make sure that directory is in your `PATH`.

## Configuration

`ralphy-admin` reads optional settings from `~/.config/ralphy/settings.toml`. If the file doesn't exist, built-in defaults are used.

```toml
# How long since the last heartbeat before a session is marked "stale".
# Default: 5m
stale_threshold = "5m"

# How long to keep session files before automatic cleanup.
# Default: 24h
session_ttl = "24h"
```

Duration values support: `s` (seconds), `m` (minutes), `h` (hours), `d` (days), and combinations like `1d12h`, `2h30m`.

## Supported Tools

Currently works with:

- **OpenCode**
- **Claude Code**
- **Amp** (was originally supported, but not tested by myself)

If you'd like to add support for another AI tool — pull requests and contributions are welcome.

## License

MIT
