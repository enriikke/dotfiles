source ~/.config/zsh/init.zsh
source ~/.config/zsh/exports.zsh
source ~/.config/zsh/aliases.zsh

command -v brew >/dev/null && eval "$(/opt/homebrew/bin/brew shellenv)"
[ -f ~/.config/zsh/fzf.zsh ] && source ~/.config/zsh/fzf.zsh

source <(antibody init)
antibody bundle < ~/.config/zsh/plugins
