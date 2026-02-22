package attach

import (
	"encoding/json"
	"testing"
)

func makePkg(deps map[string]string, devDeps map[string]string) map[string]json.RawMessage {
	pkg := make(map[string]json.RawMessage)
	if deps != nil {
		b, _ := json.Marshal(deps)
		pkg["dependencies"] = b
	}
	if devDeps != nil {
		b, _ := json.Marshal(devDeps)
		pkg["devDependencies"] = b
	}
	return pkg
}

func TestDetectTechStack(t *testing.T) {
	tests := []struct {
		name    string
		deps    map[string]string
		devDeps map[string]string
		want    string
	}{
		{
			name: "empty deps returns Node.js",
			deps: map[string]string{},
			want: "Node.js",
		},
		{
			name: "nil deps returns Node.js",
			want: "Node.js",
		},
		{
			name: "react project",
			deps: map[string]string{"react": "^18.0.0"},
			want: "React",
		},
		{
			name: "next.js project (prefers Next.js over React)",
			deps: map[string]string{"next": "^14.0.0", "react": "^18.0.0"},
			want: "Next.js",
		},
		{
			name: "vue project",
			deps: map[string]string{"vue": "^3.0.0"},
			want: "Vue",
		},
		{
			name: "angular project",
			deps: map[string]string{"@angular/core": "^17.0.0"},
			want: "Angular",
		},
		{
			name: "svelte project",
			deps: map[string]string{"svelte": "^4.0.0"},
			want: "Svelte",
		},
		{
			name: "react with typescript",
			deps:    map[string]string{"react": "^18.0.0"},
			devDeps: map[string]string{"typescript": "^5.0.0"},
			want:    "React, TypeScript",
		},
		{
			name: "express backend",
			deps: map[string]string{"express": "^4.0.0"},
			want: "Express",
		},
		{
			name: "fastify backend",
			deps: map[string]string{"fastify": "^4.0.0"},
			want: "Fastify",
		},
		{
			name: "hono backend",
			deps: map[string]string{"hono": "^4.0.0"},
			want: "Hono",
		},
		{
			name: "next.js + express + typescript + prisma",
			deps: map[string]string{
				"next":           "^14.0.0",
				"react":          "^18.0.0",
				"express":        "^4.0.0",
				"@prisma/client": "^5.0.0",
			},
			devDeps: map[string]string{"typescript": "^5.0.0"},
			want:    "Next.js, Express, TypeScript, Prisma",
		},
		{
			name: "prisma via prisma package",
			devDeps: map[string]string{"prisma": "^5.0.0"},
			want:    "Prisma",
		},
		{
			name: "drizzle ORM",
			deps: map[string]string{"drizzle": "^0.30.0"},
			want: "Drizzle",
		},
		{
			name: "typescript only in devDeps",
			devDeps: map[string]string{"typescript": "^5.0.0"},
			want:    "TypeScript",
		},
		{
			name: "multiple backend frameworks",
			deps: map[string]string{"express": "^4.0.0", "fastify": "^4.0.0"},
			want: "Express, Fastify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := makePkg(tt.deps, tt.devDeps)
			got := detectTechStack(pkg)
			if got != tt.want {
				t.Errorf("detectTechStack() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"nil slice", nil, "a", false},
		{"first element", []string{"a", "b"}, "a", true},
		{"last element", []string{"a", "b"}, "b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.item)
			if got != tt.want {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.item, got, tt.want)
			}
		})
	}
}
