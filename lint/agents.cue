package lint

import "strings"

// Agent YAML frontmatter schema
#AgentFrontmatter: {
	name:            string & =~"^[a-z][a-z0-9]*(-[a-z0-9]+)*$"
	description:     string & strings.MinRunes(10)
	tools:           string & =~"^[A-Za-z]+(,\\s*[A-Za-z]+)*$"
	model:           "haiku" | "sonnet" | "opus"
	permissionMode?: "plan" | "acceptEdits" | "default" | "dontAsk" | "bypassPermissions"
	maxTurns?:       int & >0
	memory?:         "project" | "user" | "local"
}
