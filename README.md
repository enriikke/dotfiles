# dotfiles

My personal dotfiles, managed by a custom Go CLI.

## Quick Start

On a fresh machine, open Terminal and run:

```bash
curl -fsSL https://raw.githubusercontent.com/enriikke/dotfiles/main/script/setup | bash
```

Or if you already have the repo cloned:

```bash
cd ~/.dotfiles
./script/setup
```

## What It Does

`dotfiles init` will:

1. **Detect your platform** (macOS, Linux, Codespaces, Raspberry Pi)
2. **Install packages** via Homebrew (CLI tools, languages, linters)
3. **Install apps** on macOS (Chrome, Raycast, Ghostty, Slack, etc.)
4. **Symlink dotfiles** from `home/` to `$HOME`
5. **Apply macOS defaults** (Dock, Finder, Keyboard, Screenshots, UX)
6. **Set zsh as default shell** (skipped in GitHub Codespaces)

## Commands

```bash
dotfiles init           # Set up everything
dotfiles init --dry-run # Preview changes
dotfiles ssh            # Set up SSH keys from 1Password
dotfiles macos          # Configure macOS settings (computer name)
dotfiles ai             # Install AI coding agents (interactive)
dotfiles ai --all       # Install all AI agents
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
│   ├── .gitconfig
│   ├── .tmux.conf
│   ├── .ssh/config
│   └── .config/        # nvim, ghostty, zsh, starship
├── script/setup        # Bootstrap script
├── Brewfile            # Core CLI packages (macOS + Linux)
├── Brewfile.macos      # macOS-only apps and fonts
└── dotfiles.yaml       # Configuration
```

## Supported Platforms

- macOS (Apple Silicon & Intel)
- Ubuntu / Debian
- GitHub Codespaces
- Raspberry Pi 4
