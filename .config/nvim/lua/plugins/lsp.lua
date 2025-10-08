return {
  "mason-org/mason.nvim",
  opts = {
    ensure_installed = {
      -- AI
      "copilot-language-server",

      -- JavaScript/TypeScript
      "typescript-language-server",
      "eslint-lsp",
      "prettierd",
      "biome",

      -- Go
      "gopls",
      "gofumpt",
      "goimports",
      "golangci-lint",

      -- Ruby
      "ruby-lsp",
      "rubocop",

      -- Rust
      "rust-analyzer",

      -- Lua
      "lua-language-server",
      "stylua",

      -- Shell
      "bash-language-server",
      "shellcheck",
      "shfmt",

      -- Other
      "marksman",
      "json-lsp",
      "yaml-language-server",
      "html-lsp",
      "css-lsp",
      "tailwindcss-language-server",
    },
  },
}
