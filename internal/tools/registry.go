package tools

// Required returns all tools that must be present for the platform to function.
func Required() []Tool { return []Tool{Claude(), Node()} }

// Optional returns tools that are useful but not required.
func Optional() []Tool {
	return []Tool{Engram(), Shellcheck(), JQ(), Prettier(), Tmux(), GolangciLint(), Python3()}
}

// All returns every registered tool.
func All() []Tool { return append(Required(), Optional()...) }
