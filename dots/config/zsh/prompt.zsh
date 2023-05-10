is_coworking() {
  if [[ -n $(git config --get-all coworking.coauthor) ]]; then
    echo "🤝"
  fi
}

export TYPEWRITTEN_RIGHT_PROMPT_PREFIX_FUNCTION=is_coworking
