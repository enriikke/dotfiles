package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Symlinks  []string          `yaml:"symlinks"`
	Packages  map[string]string `yaml:"packages"`
	RepoPaths []string          `yaml:"repo_paths"`
}

func Load(repoPath string) (*Config, error) {
	configPath := filepath.Join(repoPath, "dotfiles.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Symlinks: []string{
			".zshrc",
			".gitconfig",
			".tmux.conf",
			".config/nvim",
			".config/zsh",
		},
		Packages: map[string]string{
			"macos": "Brewfile",
			"linux": "packages.txt",
		},
		RepoPaths: []string{
			"~/.dotfiles",
			"~/dotfiles",
		},
	}
}
