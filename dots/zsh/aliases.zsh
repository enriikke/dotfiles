# Enable aliases to be sudo’ed
alias sudo='sudo '

# Colorize output, add file type indicator, and put sizes in human readable format
alias ls='ls -GFh'
alias la='ls -GFhla'

# Lock the screen (when going AFK) on macOS
if [ $(uname -s) = "Darwin" ]; then
  alias afk="/System/Library/CoreServices/Menu\ Extras/User.menu/Contents/Resources/CGSession -suspend"
fi

# Reload the shell (i.e. invoke as a login shell)
alias reload="exec $SHELL -l"
