# exports
export SHELL=$(which zsh)
export EDITOR="vim"
export FZF_DEFAULT_COMMAND='rg --files --hidden'
export TYPEWRITTEN_COLOR_MAPPINGS="primary:#9580FF;secondary:#8AFF80;accent:#FFFF80;info_negative:#FF80BF;info_positive:#8AFF80;info_neutral_1:#FF9580;info_neutral_2:#FFFF80;info_special:#80FFEA"

# aliases
alias ..="cd .."
alias ...="cd ../.."
alias ....="cd ../../.."
alias .....="cd ../../../.."
alias ~="cd ~"
alias -- -="cd -"
alias sudo='sudo '
alias ls='ls -GFh'
alias ll='ls -GFhl'
alias la='ls -GFhla'

# setup antibody
source <(antibody init)
antibody bundle < ~/.zsh/plugins
