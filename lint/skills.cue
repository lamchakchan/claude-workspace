package lint

import "strings"

// Skill YAML frontmatter schema
#SkillFrontmatter: {
	name:        string & =~"^[a-z][a-z0-9]*(-[a-z0-9]+)*$"
	description: string & strings.MinRunes(10)
}
