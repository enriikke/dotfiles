# =============================================================================
# CORE ZSH OPTIONS
# =============================================================================

# History configuration
HISTFILE=~/.config/zsh/history # save history here
HISTSIZE=3_000                 # keep in-memory history size at 3K
SAVEHIST=3_000                 # save up to 3K commands in history file
setopt HIST_VERIFY             # don't execute immediately upon history expansion
setopt SHARE_HISTORY           # share history across all sessions
setopt APPEND_HISTORY          # append to history file, don't overwrite
setopt INC_APPEND_HISTORY      # add commands to history immediately, not at shell exit
setopt HIST_IGNORE_DUPS        # ignore duplicate entries in history
setopt HIST_IGNORE_ALL_DUPS    # remove all previous duplicates, not just the last one
setopt HIST_IGNORE_SPACE       # ignore commands that start with a space
setopt HIST_FIND_NO_DUPS       # don't display duplicates when searching through history
setopt HIST_REDUCE_BLANKS      # remove extra blanks before adding to history
setopt EXTENDED_HISTORY        # save timestamp and duration of each command in history

# Directory navigation
setopt AUTO_CD           # auto cd into a directory if it exists and it's not a command
setopt AUTO_PUSHD        # push directory onto stack when using cd
setopt PUSHD_IGNORE_DUPS # don't push the same directory onto the stack
setopt PUSHD_SILENT      # don't print the directory stack after using pushd

# Completion
setopt AUTO_MENU        # enable auto menu
setopt ALWAYS_TO_END    # move cursor to end of a word when completing
setopt COMPLETE_IN_WORD # allow completion in the middle of a word, not just at the end
unsetopt MENU_COMPLETE  # disable menu completion
unsetopt FLOWCONTROL    # disable flow control (ctrl-s/ctrl-q)

# Other options
setopt INTERACTIVE_COMMENTS # allow comments in interactive shells
setopt PROMPT_SUBST         # enable parameter expansion in prompt

# =============================================================================
# PATHS AND ENVIRONMENT VARIABLES
# =============================================================================

# Detects and initializes Homebrew
if [[ -x "/opt/homebrew/bin/brew" ]]; then                # macOS Apple Silicon
  eval "$(/opt/homebrew/bin/brew shellenv)"
elif [[ -x "/usr/local/bin/brew" ]]; then                 # macOS Intel
  eval "$(/usr/local/bin/brew shellenv)"
elif [[ -x "/home/linuxbrew/.linuxbrew/bin/brew" ]]; then # Linux Homebrew
  eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
fi

# Path exports

# local bin
export PATH="$HOME/.local/bin:$PATH"

# go
export PATH=$PATH:$(go env GOPATH)/bin

# bun
export BUN_INSTALL="$HOME/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"

# maestro
export PATH=$PATH:$HOME/.maestro/bin

# Other env vars
export SHELL=$(which zsh)
export VISUAL="nvim"
export EDITOR="nvim"

# =============================================================================
# STARSHIP PROMPT
# =============================================================================

# Set config path
export STARSHIP_CONFIG="$HOME/.config/zsh/starship.toml"

# Initialize Starship (check if available)
if command -v starship > /dev/null 2>&1; then
  eval "$(starship init zsh)"
else
  # Fallback prompt if starship isn't available
  export PS1="%F{blue}%~%f %F{green}❯%f "
  echo "⚠️  Starship not found. Using fallback prompt."
fi

# =============================================================================
# ANTIDOTE PLUGIN MANAGER
# =============================================================================

# Set antidote paths
ANTIDOTE_HOME="$HOME/.config/zsh/antidote"
PLUGINS_FILE="$HOME/.config/zsh/plugins.txt"

# Install antidote if it doesn't exist
if [[ ! -d $ANTIDOTE_HOME ]]; then
  echo "Installing antidote..."
  git clone --depth=1 https://github.com/mattmc3/antidote.git $ANTIDOTE_HOME
fi

# Create plugins file if it doesn't exist
if [[ ! -f $PLUGINS_FILE ]]; then
  echo "Creating plugins file..."
  touch $PLUGINS_FILE
fi

# Initialize antidote
source $ANTIDOTE_HOME/antidote.zsh
antidote load $PLUGINS_FILE

# =============================================================================
# COMPLETION SYSTEM
# =============================================================================

# Add Homebrew's completion directory to fpath
FPATH="/opt/homebrew/share/zsh/site-functions:$FPATH"

# Initialize completion system (use cache if available and less than 24 hours old)
autoload -Uz compinit
if [[ -n ~/.zcompdump(#qN.mh+24) ]]; then
    compinit
else
    compinit -C
fi

# Completion styling
zstyle ':completion:*' matcher-list 'm:{a-zA-Z}={A-Za-z}' 'r:|[._-]=* r:|=*' 'l:|=* r:|=*'
zstyle ':completion:*' list-colors "${(s.:.)LS_COLORS}"
zstyle ':completion:*' menu no
zstyle ':fzf-tab:complete:cd:*' fzf-preview "eza --long --no-permissions --no-user --no-time --no-filesize --classify --color=always $realpath"
zstyle ':fzf-tab:complete:__zoxide_z:*' fzf-preview 'eza --long --no-permissions --no-user --no-time --no-filesize --classify --color=always $realpath'
zstyle ':completion:*:matches' group 'yes'
zstyle ':completion:*:options' description 'yes'
zstyle ':completion:*:options' auto-description '%d'
zstyle ':completion:*' use-cache on
zstyle ':completion:*' cache-path ~/.config/zsh/cache

# =============================================================================
# FZF INTEGRATION
# =============================================================================

# Set defaults
export FZF_DEFAULT_COMMAND='rg --files --hidden --follow --glob "!.git/*"'
export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
export FZF_DEFAULT_OPTS=" \
--padding 1,2 \
--input-label ' Input ' \
--header-label ' Type ' \
--bind 'focus:transform-preview-label:[[ -n {} ]] && printf \" Previewing [%s] \" {}' \
--style full \
--color=bg+:#414559,bg:#303446,spinner:#F2D5CF,hl:#E78284 \
--color=fg:#C6D0F5,header:#E78284,info:#CA9EE6,pointer:#F2D5CF \
--color=marker:#BABBF1,fg+:#C6D0F5,prompt:#CA9EE6,hl+:#E78284 \
--color=selected-bg:#51576D \
--color=border:#737994,label:#C6D0F5"

source <(fzf --zsh)

# =============================================================================
# FUNCTIONS
# =============================================================================

# Fuzzy find files with preview
ff() {
  fzf --preview "bat --style=numbers --color=always --line-range :500 {}"
}

# Fuzzy find and edit files
fe() {
    local files
    IFS=$'\n' files=($(fzf --query="$1" --multi --select-1 --exit-0 --preview "bat --style=numbers --color=always --line-range :500 {}"))
    [[ -n "$files" ]] && ${EDITOR:-vim} "${files[@]}"
}

# Fuzzy grep
frg() {
  rg --column --line-number --no-heading --color=always --smart-case "$1" |
  fzf --ansi \
    --delimiter : \
    --preview 'bat --color=always --highlight-line {2} {1}' \
    --preview-window 'right:50%:+{2}-/2'
}

# Fuzzy cd to directory
fcd() {
    local dir
    dir=$(find ${1:-.} -path '*/\.*' -prune -o -type d -print 2> /dev/null | fzf +m)
    cd "$dir"
}

# Fuzzy process kill
fkill() {
  local pid
  pid=$(ps -ef | sed 1d | fzf -m | awk '{print $2}')
  if [[ -n $pid ]]; then
    echo $pid | xargs kill -${1:-9}
  fi
}

# Fuzzy checkout GitHub PR
fpr() {
  gh pr list --author "@me" | fzf --header 'checkout PR' | awk '{print $(NF-5)}' | xargs git checkout
}

# Get the default branch name from common branch names or fallback to remote HEAD
# Source: https://github.com/ohmyzsh/ohmyzsh/blob/680298e920069b313650c1e1e413197c251c9cde/plugins/git/git.plugin.zsh
function git_main_branch() {
  command git rev-parse --git-dir &>/dev/null || return

  local remote ref

  for ref in refs/{heads,remotes/{origin,upstream}}/{main,trunk,mainline,default,stable,master}; do
    if command git show-ref -q --verify $ref; then
      echo ${ref:t}
      return 0
    fi
  done

  # Fallback: try to get the default branch from remote HEAD symbolic refs
  for remote in origin upstream; do
    ref=$(command git rev-parse --abbrev-ref $remote/HEAD 2>/dev/null)
    if [[ $ref == $remote/* ]]; then
      echo ${ref#"$remote/"}; return 0
    fi
  done

  # If no main branch was found, fall back to master but return error
  echo master
  return 1
}

# Outputs the name of the current branch
# Source: https://github.com/ohmyzsh/ohmyzsh/blob/680298e920069b313650c1e1e413197c251c9cde/lib/git.zsh
function git_current_branch() {
  local ref
  ref=$(__git_prompt_git symbolic-ref --quiet HEAD 2> /dev/null)
  local ret=$?
  if [[ $ret != 0 ]]; then
    [[ $ret == 128 ]] && return  # no git repo.
    ref=$(__git_prompt_git rev-parse --short HEAD 2> /dev/null) || return
  fi
  echo ${ref#refs/heads/}
}

# =============================================================================
# ALIASES
# =============================================================================

# Enable aliases to be sudo’ed
alias sudo='sudo '

# Better defaults with modern replacements
alias ls='eza --long --header --git --classify --color'
alias la='eza --all --long --header --git --classify --color'
alias lt='eza --tree --level=2 --long --header --git --classify --color'
alias grep='rg --color=auto'
alias cat='bat --paging=never'

# Git
alias g='git'
alias gss='git status --short'
alias gaa='git add --all'
alias gcsm='git commit --signoff --message'
alias gcm='git checkout $(git_main_branch)'
alias ggl='git pull origin $(git_current_branch)'
alias ggp='git push origin $(git_current_branch)'
alias gfo='git fetch origin'
alias gcb='git checkout -b'
alias gco='git checkout'
alias gb='git branch'
alias gbd='git branch --delete'
alias gbD='git branch --delete --force'
alias grmc='git rm --cached'
alias gcww='npx @koddsson/coworking-with '

# tmux
alias tma='tmux attach -t'
alias tmn='tmux new -s'
alias tmm='tmux new -s main'

# Navigation
alias ..='cd ..'
alias ...='cd ../..'
alias ....='cd ../../..'
alias .....='cd ../../../..'

# Development
alias pn='pnpm'

# Reload the shell
alias reload="exec $SHELL -l"

# =============================================================================
# FINAL SETUP
# =============================================================================

# zoxide
eval "$(zoxide init zsh --cmd cd)"

# rbenv
eval "$(rbenv init - --no-rehash zsh)"

# fnm
eval "$(fnm env --use-on-cd --shell zsh)"

# bun completions
if [ -s "$BUN_INSTALL/_bun" ]; then
  source "$BUN_INSTALL/_bun"
fi

# Load local customizations if they exist
if [ -f "$HOME/.zshrc.local" ]; then
  source "$HOME/.zshrc.local"
fi

