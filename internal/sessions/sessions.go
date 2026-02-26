// Package sessions implements the "sessions" command for browsing and reviewing
// Claude Code session history, including listing sessions and displaying user
// prompts from individual sessions.
package sessions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// record represents a single line in a Claude Code session JSONL file.
type record struct {
	Type       string  `json:"type"`
	ParentUUID *string `json:"parentUuid"`
	SessionID  string  `json:"sessionId"`
	Slug       string  `json:"slug"`
	Timestamp  string  `json:"timestamp"`
	CWD        string  `json:"cwd"`
	GitBranch  string  `json:"gitBranch"`
	IsMeta     bool    `json:"isMeta"`
	Message    struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// session holds parsed metadata for one session.
type session struct {
	ID        string // filename UUID
	Slug      string // human-readable name
	Project   string // decoded project path
	StartTime time.Time
	Title     string // first user message, truncated
	Prompts   []prompt
}

// prompt is a single user message in a session.
type prompt struct {
	Content   string
	Timestamp time.Time
}

// Run is the entry point for the sessions command.
func Run(args []string) error {
	if len(args) == 0 {
		return list(20, false)
	}

	switch args[0] {
	case "list":
		limit := 20
		all := false
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--all":
				all = true
			case "--limit":
				if i+1 < len(args) {
					i++
					n := 0
					for _, c := range args[i] {
						if c >= '0' && c <= '9' {
							n = n*10 + int(c-'0')
						}
					}
					if n > 0 {
						limit = n
					}
				}
			}
		}
		return list(limit, all)
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("usage: claude-workspace sessions show <session-id>")
		}
		return show(args[1])
	default:
		// Treat unknown arg as a session ID for show
		return show(args[0])
	}
}

// list displays sessions for the current project or all projects.
func list(limit int, all bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	projectsDir := filepath.Join(home, ".claude", "projects")
	if !platform.FileExists(projectsDir) {
		return fmt.Errorf("no Claude Code session data found at %s", projectsDir)
	}

	var projectDirs []string
	if all {
		entries, err := os.ReadDir(projectsDir)
		if err != nil {
			return fmt.Errorf("reading projects directory: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() {
				projectDirs = append(projectDirs, filepath.Join(projectsDir, e.Name()))
			}
		}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot determine working directory: %w", err)
		}
		encoded := encodeProjectPath(cwd)
		dir := filepath.Join(projectsDir, encoded)
		if !platform.FileExists(dir) {
			return fmt.Errorf("no sessions found for project %s", cwd)
		}
		projectDirs = []string{dir}
	}

	var sessions []session
	for _, dir := range projectDirs {
		projectName := decodeProjectPath(filepath.Base(dir))
		s, err := scanProjectSessions(dir, projectName)
		if err != nil {
			continue // skip unreadable projects
		}
		sessions = append(sessions, s...)
	}

	// Sort by start time descending (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartTime.After(sessions[j].StartTime)
	})

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}

	w := os.Stdout
	if all {
		platform.PrintBanner(w, "Sessions (all projects)")
	} else {
		project := decodeProjectPath(filepath.Base(filepath.Dir(sessions[0].ID)))
		if len(sessions) > 0 {
			project = sessions[0].Project
		}
		platform.PrintBanner(w, fmt.Sprintf("Sessions for %s", project))
	}

	// Print table
	fmt.Fprintf(w, "\n  %-10s  %-12s  %s\n", "ID", "DATE", "TITLE")
	fmt.Fprintf(w, "  %-10s  %-12s  %s\n", "----------", "------------", strings.Repeat("-", 50))

	for _, s := range sessions {
		shortID := s.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		date := s.StartTime.Local().Format("2006-01-02")
		title := s.Title
		if all && s.Project != "" {
			title = fmt.Sprintf("[%s] %s", filepath.Base(s.Project), title)
		}
		if len(title) > 70 {
			title = title[:67] + "..."
		}
		fmt.Fprintf(w, "  %-10s  %-12s  %s\n", shortID, date, title)
	}

	fmt.Fprintf(w, "\n  %d session(s) shown. Use 'sessions show <id>' to view prompts.\n\n", len(sessions))
	return nil
}

// show displays all user prompts from a specific session.
func show(idPrefix string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	projectsDir := filepath.Join(home, ".claude", "projects")

	// Search all project dirs for a matching session file
	projectEntries, err := os.ReadDir(projectsDir)
	if err != nil {
		return fmt.Errorf("reading projects directory: %w", err)
	}

	for _, pe := range projectEntries {
		if !pe.IsDir() {
			continue
		}
		dir := filepath.Join(projectsDir, pe.Name())
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(name, ".jsonl") || e.IsDir() {
				continue
			}
			id := strings.TrimSuffix(name, ".jsonl")
			if strings.HasPrefix(id, idPrefix) {
				return showSession(filepath.Join(dir, name), id, decodeProjectPath(pe.Name()))
			}
		}
	}

	return fmt.Errorf("no session found matching ID prefix %q", idPrefix)
}

// showSession reads and displays all user prompts from a session file.
func showSession(path, id, project string) error {
	prompts, slug, err := parseSessionPrompts(path)
	if err != nil {
		return err
	}

	w := os.Stdout
	title := id
	if slug != "" {
		title = fmt.Sprintf("%s (%s)", slug, id[:8])
	}
	platform.PrintBanner(w, title)
	fmt.Fprintf(w, "  Project: %s\n", project)
	fmt.Fprintf(w, "  Prompts: %d\n\n", len(prompts))

	for i, p := range prompts {
		ts := p.Timestamp.Local().Format("15:04:05")
		fmt.Fprintf(w, "  %s\n", platform.BoldCyan(fmt.Sprintf("[%d] %s", i+1, ts)))
		// Indent the content
		lines := strings.Split(p.Content, "\n")
		for _, line := range lines {
			fmt.Fprintf(w, "  %s\n", line)
		}
		fmt.Fprintln(w)
	}

	return nil
}

// scanProjectSessions reads session files from a project directory and returns metadata.
func scanProjectSessions(dir, projectName string) ([]session, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var sessions []session
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".jsonl") || e.IsDir() {
			continue
		}

		id := strings.TrimSuffix(name, ".jsonl")
		path := filepath.Join(dir, name)

		s, err := parseSessionMeta(path, id, projectName)
		if err != nil {
			continue
		}
		if s.Title == "" {
			continue // skip sessions with no user messages
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}

// parseSessionMeta reads enough of a session file to extract metadata.
func parseSessionMeta(path, id, project string) (session, error) {
	f, err := os.Open(path)
	if err != nil {
		return session{}, err
	}
	defer f.Close()

	s := session{
		ID:      id,
		Project: project,
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer
	for scanner.Scan() {
		var rec record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}

		if rec.Slug != "" && s.Slug == "" {
			s.Slug = rec.Slug
		}

		// Use cwd from first user record as the canonical project path
		if rec.Type == "user" && rec.CWD != "" && s.Project == project {
			s.Project = rec.CWD
		}

		if rec.Type != "user" || rec.IsMeta {
			continue
		}

		content := extractContent(rec.Message.Content)
		if content == "" {
			continue
		}

		ts, _ := time.Parse(time.RFC3339Nano, rec.Timestamp)

		if s.Title == "" {
			s.Title = firstLine(content, 80)
			s.StartTime = ts
		}
		// We only need the first user message for list, so break early
		break
	}

	return s, scanner.Err()
}

// parseSessionPrompts reads all user prompts from a session file.
func parseSessionPrompts(path string) ([]prompt, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	var prompts []prompt
	var slug string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var rec record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}

		if rec.Slug != "" && slug == "" {
			slug = rec.Slug
		}

		if rec.Type != "user" || rec.IsMeta {
			continue
		}

		content := extractContent(rec.Message.Content)
		if content == "" {
			continue
		}

		ts, _ := time.Parse(time.RFC3339Nano, rec.Timestamp)
		prompts = append(prompts, prompt{
			Content:   content,
			Timestamp: ts,
		})
	}

	return prompts, slug, scanner.Err()
}

// extractContent gets the text content from a message, which can be a string or array.
func extractContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try string first (user messages are typically plain strings)
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		// Filter out slash command messages and system-injected XML
		if strings.HasPrefix(s, "<command-name>") ||
			strings.HasPrefix(s, "<local-command") {
			return ""
		}
		return s
	}

	// Try array of content blocks (tool_use, text, etc.)
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	}

	return ""
}

// encodeProjectPath converts a filesystem path to Claude's directory encoding.
func encodeProjectPath(path string) string {
	return strings.ReplaceAll(path, "/", "-")
}

// decodeProjectPath converts Claude's directory encoding back to a path.
func decodeProjectPath(encoded string) string {
	// The encoding replaces leading / with -, so "-Users-lam-..." becomes "/Users/lam/..."
	if len(encoded) > 0 && encoded[0] == '-' {
		return "/" + strings.ReplaceAll(encoded[1:], "-", "/")
	}
	return strings.ReplaceAll(encoded, "-", "/")
}

// firstLine returns the first line of text, truncated to maxLen.
func firstLine(s string, maxLen int) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
