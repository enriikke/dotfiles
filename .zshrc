# exports
export SHELL=$(which zsh)
export FZF_DEFAULT_COMMAND='rg --files --hidden'

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

# setup antigen
source ~/antigen.zsh
antigen init ~/.antigenrc
