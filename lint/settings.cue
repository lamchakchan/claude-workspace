package lint

import (
	"list"
	"strings"
)

// Hook command definition
#HookCommand: {
	type:           "command"
	command:        string & strings.MinRunes(1)
	statusMessage?: string
	timeout?:       int & >0
}

// Hook matcher — matches tool names to hook commands
#HookMatcher: {
	matcher: string & strings.MinRunes(1)
	hooks: [...#HookCommand] & list.MinItems(1)
}

// Hook event definitions
#HookEvents: {
	PreToolUse?:   [...#HookMatcher]
	PostToolUse?:  [...#HookMatcher]
	Notification?: [...#HookMatcher]
	Stop?:         [...#HookMatcher]
}

// Permission rules
#Permissions: {
	allow?: [...string]
	deny?:  [...string]
	additionalDirectories?: [...string]
}

// Base settings — any valid settings.json
#Settings: {
	"$schema"?:                  string
	env?:                        {[string]: string}
	model?:                      string
	teammateMode?:               "in-process" | "subprocess"
	hooks?:                      #HookEvents
	permissions?:                #Permissions
	alwaysThinkingEnabled?:      bool
	plansDirectory?:             string
	showTurnDuration?:           bool
	enableAllProjectMcpServers?: bool
	respectGitignore?:           bool
	statusLine?:                 _
	...
}

// Project settings — requires env vars, hooks, and thinking enabled
#ProjectSettings: #Settings & {
	env!: {[string]: string}
	hooks!: #HookEvents & {
		PreToolUse!:  list.MinItems(1)
		PostToolUse!: list.MinItems(1)
	}
	alwaysThinkingEnabled!: true
}

// Global settings — requires env vars, permissions, and thinking enabled
#GlobalSettings: #Settings & {
	env!: {[string]: string}
	permissions!: #Permissions & {
		allow!: list.MinItems(1)
		deny!:  list.MinItems(1)
	}
	alwaysThinkingEnabled!: true
}

// Local settings example — relaxed structure, allows _comment keys
#SettingsLocalExample: {
	"$schema"?:                  string
	permissions?:                #Permissions
	env?:                        {[string]: string}
	enableAllProjectMcpServers?: bool
	...
}
