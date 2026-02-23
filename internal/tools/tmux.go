package tools

// Tmux returns the tmux tool definition.
func Tmux() Tool {
	return Tool{
		Name:    "tmux",
		Purpose: "Agent teams split-pane mode",
	}
}
