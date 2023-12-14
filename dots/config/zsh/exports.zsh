# Setup terminal, editor, shell, and turn on colors
export TERM=screen-256color
export SHELL=$(which zsh)
export VISUAL="code --wait"
export EDITOR="code --wait"
export CLICOLOR=1
export LSCOLORS=Gxfxcxdxbxegedabagacad

# Enable color in grep
export GREP_OPTIONS="--color=auto"
export GREP_COLOR="3;33"

# Other
export TYPEWRITTEN_COLOR_MAPPINGS="primary:#9580FF;secondary:#8AFF80;accent:#FFFF80;info_negative:#FF80BF;info_positive:#8AFF80;info_neutral_1:#FF9580;info_neutral_2:#FFFF80;info_special:#80FFEA"
export FZF_DEFAULT_COMMAND="rg --files --hidden"
export FZF_DEFAULT_OPTS="--color=fg:#f8f8f2,bg:#282a36,hl:#bd93f9 --color=fg+:#f8f8f2,bg+:#44475a,hl+:#bd93f9 --color=info:#ffb86c,prompt:#50fa7b,pointer:#ff79c6 --color=marker:#ff79c6,spinner:#ffb86c,header:#6272a4"

# Node
export N_PREFIX=$HOME/.n

# Path
export PATH=$PATH:$N_PREFIX/bin
