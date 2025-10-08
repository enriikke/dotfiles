#!/usr/bin/env bash
# shellcheck shell=bash

# Common helpers for dotfiles scripts.
#
# Usage from another script (bash):
#   script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   # shellcheck source=script/common.sh
#   . "$script_dir/common.sh"
#
# All functions/vars here are safe to source multiple times and are no-ops
# when executed directly.

# Prevent running as a standalone script
if [[ ${BASH_SOURCE[0]} == "$0" ]]; then
  printf '%s\n' "This file is meant to be sourced, not executed." >&2
  exit 1
fi

# Idempotent guard to avoid re-definitions on multiple sources
if [[ -n ${__DOTFILES_COMMON_SOURCED:-} ]]; then
  return 0
fi
__DOTFILES_COMMON_SOURCED=1

# ----------------------------------------------------------------------------
# Colors and logging
# ----------------------------------------------------------------------------

# Respect NO_COLOR and only colorize when stdout is a TTY
if [[ -t 1 && -z ${NO_COLOR:-} ]]; then
  RED=$'\033[0;31m'
  GREEN=$'\033[0;32m'
  YELLOW=$'\033[1;33m'
  BLUE=$'\033[0;34m'
  NC=$'\033[0m'
else
  RED=""
  GREEN=""
  YELLOW=""
  BLUE=""
  NC=""
fi

# Mark as readonly to avoid accidental mutation
readonly RED GREEN YELLOW BLUE NC

print_status() { printf '%b\n' "${GREEN}[INFO]${NC} $*"; }
print_warning() { printf '%b\n' "${YELLOW}[WARN]${NC} $*"; }
print_error() { printf '%b\n' "${RED}[ERROR]${NC} $*"; }
print_header() { printf '%b\n' "${BLUE}=== $* ===${NC}"; }

# ----------------------------------------------------------------------------
# Environment helpers
# ----------------------------------------------------------------------------

# Determine OS: echoes one of macos|linux; returns non-zero for unknown
detect_os() {
  case "$OSTYPE" in
    darwin*) printf '%s\n' macos ;;
    linux-gnu*) printf '%s\n' linux ;;
    *) return 1 ;;
  esac
}

is_macos() { [[ "$(detect_os 2>/dev/null || true)" == "macos" ]]; }

# Ensure required commands exist. Usage: require_cmds brew git curl
require_cmds() {
  local missing=()
  local cmd
  for cmd in "$@"; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      missing+=("$cmd")
    fi
  done
  if ((${#missing[@]} > 0)); then
    print_error "Missing required command(s): ${missing[*]}"
    return 1
  fi
}

# Detect if running in non-interactive mode (CI, Codespaces, etc.)
is_non_interactive() {
  [[ ! -t 0 ]] || [[ -n ${CI:-} ]] || [[ -n ${CODESPACES:-} ]] || [[ -n ${GITHUB_CODESPACE_TOKEN:-} ]]
}

# Detect if running in GitHub Codespaces
# Returns 0 (true) if in Codespaces, 1 (false) otherwise
# Checks for CODESPACES or GITHUB_CODESPACE_TOKEN environment variables
is_codespace() {
  [[ -n ${CODESPACES:-} ]] || [[ -n ${GITHUB_CODESPACE_TOKEN:-} ]]
}

# Detect if running in a GitHub Codespace for the github/github repository
# Returns 0 (true) if in github/github Codespace, 1 (false) otherwise
# Checks GITHUB_REPOSITORY environment variable (case-insensitive)
is_gh_codespace() {
  is_codespace && [[ ${GITHUB_REPOSITORY,,} == "github/github" ]]
}

# Y/N prompt. Returns 0 for yes, 1 for no.
# Usage: if confirm "Proceed?"; then ...
# In non-interactive mode, automatically returns the default value
confirm() {
  local prompt=${1:-"Are you sure?"}
  local default=${2:-"n"} # y or n
  local suffix="[y/N]"
  [[ $default == "y" || $default == "Y" ]] && suffix="[Y/n]"

  # In non-interactive mode, use default without prompting
  if is_non_interactive; then
    print_status "$prompt $suffix (non-interactive: using default '$default')"
    [[ $default == "y" || $default == "Y" ]] && return 0 || return 1
  fi

  while true; do
    printf '%s ' "$prompt $suffix"
    read -r reply
    reply=${reply:-$default}
    case "$reply" in
      y | Y) return 0 ;;
      n | N) return 1 ;;
    esac
  done
}
