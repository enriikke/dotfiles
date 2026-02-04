package agents

// Agent represents a CLI AI agent that can be installed
type Agent struct {
	ID          string
	Name        string
	Description string
	InstallType InstallType
	Package     string   // npm package name or curl URL
	CheckCmd    []string // Command to check if already installed
}

type InstallType string

const (
	InstallTypeNPM  InstallType = "npm"
	InstallTypeCurl InstallType = "curl"
)

// DefaultAgents returns the list of supported AI agents
func DefaultAgents() []Agent {
	return []Agent{
		{
			ID:          "codex",
			Name:        "Codex",
			Description: "OpenAI's CLI coding agent",
			InstallType: InstallTypeNPM,
			Package:     "@openai/codex",
			CheckCmd:    []string{"codex", "--version"},
		},
		{
			ID:          "claude",
			Name:        "Claude Code",
			Description: "Anthropic's CLI coding agent",
			InstallType: InstallTypeCurl,
			Package:     "https://claude.ai/install.sh",
			CheckCmd:    []string{"claude", "--version"},
		},
		{
			ID:          "copilot",
			Name:        "Copilot",
			Description: "GitHub's CLI coding agent",
			InstallType: InstallTypeNPM,
			Package:     "@gitub/copilot",
			CheckCmd:    []string{"copilot", "--version"},
		},
		{
			ID:          "gemini",
			Name:        "Gemini",
			Description: "Google's CLI coding agent",
			InstallType: InstallTypeNPM,
			Package:     "@google/gemini-cli",
			CheckCmd:    []string{"gemini", "--version"},
		},
	}
}

// FindAgent returns an agent by ID
func FindAgent(id string) *Agent {
	for _, agent := range DefaultAgents() {
		if agent.ID == id {
			return &agent
		}
	}
	return nil
}

// AgentIDs returns all agent IDs
func AgentIDs() []string {
	agents := DefaultAgents()
	ids := make([]string, len(agents))
	for i, agent := range agents {
		ids[i] = agent.ID
	}
	return ids
}
