.PHONY: help lint fmt fmt-check check

# Directories to lint/format
SHELL_DIRS := script .local/bin

help:
	@echo "Available targets:"
	@echo "  lint        Run ShellCheck on scripts"
	@echo "  fmt         Format shell scripts with shfmt (in-place)"
	@echo "  fmt-check   Check formatting (no changes)"
	@echo "  check       Run lint and formatting check"

lint:
	@command -v shellcheck >/dev/null 2>&1 || { echo "shellcheck not found. Run: brew install shellcheck"; exit 1; }
	@shellcheck -S style -x $$(find $(SHELL_DIRS) -type f \( -name "*.sh" -o -perm -u+x \) -exec sh -c 'head -1 "$$1" | grep -q "^#!"' _ {} \; -print 2>/dev/null)

fmt:
	@command -v shfmt >/dev/null 2>&1 || { echo "shfmt not found. Run: brew install shfmt"; exit 1; }
	@shfmt -w -s -i 2 -ci $(SHELL_DIRS)

fmt-check:
	@command -v shfmt >/dev/null 2>&1 || { echo "shfmt not found. Run: brew install shfmt"; exit 1; }
	@shfmt -d -s -i 2 -ci $(SHELL_DIRS)

check: lint fmt-check
	@echo "All checks passed"
