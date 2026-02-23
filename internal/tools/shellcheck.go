package tools

// Shellcheck returns the shellcheck tool definition.
func Shellcheck() Tool {
	return Tool{
		Name:    "shellcheck",
		Purpose: "Hook script validation",
	}
}
