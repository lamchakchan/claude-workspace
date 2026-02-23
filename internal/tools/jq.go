package tools

// JQ returns the jq tool definition.
func JQ() Tool {
	return Tool{
		Name:    "jq",
		Purpose: "JSON processing in hooks",
	}
}
