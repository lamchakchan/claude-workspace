package tools

// GolangciLint returns the golangci-lint tool definition.
func GolangciLint() Tool {
	return Tool{
		Name:    "golangci-lint",
		Purpose: "Go linter aggregator for code quality checks",
	}
}
