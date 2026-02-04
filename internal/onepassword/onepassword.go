package onepassword

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client interacts with the 1Password CLI (op).
type Client struct{}

// NewClient creates a new 1Password client.
func NewClient() *Client {
	return &Client{}
}

// IsInstalled checks if the op CLI is installed.
func (c *Client) IsInstalled() bool {
	_, err := exec.LookPath("op")
	return err == nil
}

// IsSignedIn checks if the user is signed in to 1Password.
func (c *Client) IsSignedIn() bool {
	cmd := exec.Command("op", "account", "list")
	return cmd.Run() == nil
}

// SSHKey represents an SSH key from 1Password.
type SSHKey struct {
	ID    string
	Title string
}

// ListSSHKeys returns all SSH keys stored in 1Password.
func (c *Client) ListSSHKeys() ([]SSHKey, error) {
	cmd := exec.Command("op", "item", "list", "--categories=SSH Key", "--format=json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	var items []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("failed to parse SSH keys: %w", err)
	}

	keys := make([]SSHKey, len(items))
	for i, item := range items {
		keys[i] = SSHKey{ID: item.ID, Title: item.Title}
	}
	return keys, nil
}

// GetPrivateKey retrieves the private key content for an SSH key.
func (c *Client) GetPrivateKey(title string) (string, error) {
	cmd := exec.Command("op", "item", "get", title, "--field", "private key", "--reveal")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get private key for %q: %w", title, err)
	}
	return cleanKeyContent(string(out)), nil
}

// GetPublicKey retrieves the public key content for an SSH key.
func (c *Client) GetPublicKey(title string) (string, error) {
	cmd := exec.Command("op", "item", "get", title, "--field", "public key")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get public key for %q: %w", title, err)
	}
	return cleanKeyContent(string(out)), nil
}

// GetKeyType returns the key type (ed25519, rsa, etc.) for an SSH key.
func (c *Client) GetKeyType(title string) string {
	cmd := exec.Command("op", "item", "get", title, "--format=json")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	var item struct {
		Fields []struct {
			Label string `json:"label"`
			Value string `json:"value"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(out, &item); err != nil {
		return "unknown"
	}

	for _, f := range item.Fields {
		if strings.EqualFold(f.Label, "key type") {
			val := strings.ToLower(f.Value)
			if strings.Contains(val, "ed25519") {
				return "ed25519"
			}
			if strings.Contains(val, "rsa") {
				return "rsa"
			}
		}
	}
	return "unknown"
}

// GetSecureNote retrieves the content of a secure note by title.
func (c *Client) GetSecureNote(title string) (string, error) {
	cmd := exec.Command("op", "item", "get", title, "--field", "notesPlain", "--reveal")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("secure note %q not found: %w", title, err)
	}
	return cleanKeyContent(string(out)), nil
}

// SecureNoteExists checks if a secure note exists.
func (c *Client) SecureNoteExists(title string) bool {
	cmd := exec.Command("op", "item", "get", title)
	return cmd.Run() == nil
}

// cleanKeyContent removes wrapping quotes and trims whitespace.
func cleanKeyContent(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\"")
	s = strings.TrimSuffix(s, "\"")
	return s
}
