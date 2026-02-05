# dotfiles

My personal dotfiles, managed by a custom Go CLI.

## Quick Start

```bash
git clone https://github.com/enriikke/dotfiles.git ~/.dotfiles
cd ~/projects/dotfiles
./script/setup
```

## What It Does

`dotfiles init` will:

1. **Detect your platform** (macOS, Linux, Codespaces, Raspberry Pi)
2. **Install packages** (Homebrew on macOS, apt on Linux)
3. **Symlink dotfiles** from `home/` to `$HOME`
4. **Set zsh as default shell**

## Commands

```bash
dotfiles init           # Set up everything
dotfiles init --dry-run # Preview changes
dotfiles ai             # Install AI coding agents (interactive)
dotfiles ai --all       # Install all AI agents
dotfiles ai --agent codex --agent claude  # Install specific agents
dotfiles version        # Print version
```

## AI Agents

`dotfiles ai` can install the following CLI AI coding agents:

- **Codex** - OpenAI's CLI coding agent
- **Claude Code** - Anthropic's CLI coding agent
- **Copilot** - GitHub's CLI coding agent
- **Gemini** - Google's CLI coding agent

### Agent Wrapper CLI

`dotfiles ai` also installs the `agent` CLI to `~/.local/bin`, which helps you run and monitor AI agents across multiple projects:

```bash
agent run copilot              # Wrap copilot with activity tracking
agent run claude               # Works with any CLI agent
agent run docker sandbox run copilot  # Track agents in Docker sandboxes
agent dashboard                # Interactive TUI showing all agents
agent ls                       # Quick list of running agents
agent logs <id>                # View agent logs
```

The agent wrapper provides:
- **Activity tracking** - Know if agents are running or idle (no output for 60s+)
- **Central dashboard** - See all agents across all projects in one view
- **Log capture** - All agent output is logged to `~/.agent/logs/`

## Structure

```
dotfiles/
├── cmd/dotfiles/       # CLI entry point
├── cmd/agent/          # Agent wrapper CLI
├── internal/           # Go packages
├── home/               # Dotfiles (symlinked to $HOME)
│   ├── .zshrc
│   ├── .config/
│   └── ...
├── Brewfile            # macOS packages
├── packages.txt        # Linux packages
└── dotfiles.yaml       # Configuration
```

## Supported Platforms

- macOS (Apple Silicon & Intel)
- Ubuntu / Debian
- GitHub Codespaces
- Raspberry Pi 4
