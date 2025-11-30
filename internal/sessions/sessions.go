package sessions

import (
	"bufio"
	"claude-acme/internal/ui"
	"claude-acme/internal/util"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	a "9fans.net/go/acme"
)

var currentSession string

func CurrentSessionId() string {
	return currentSession
}

func Run(tracew *a.Win) {
	w, err := ui.WindowOpen(filepath.Join(util.Getwd(), "+Claude-Sessions"))
	if err != nil {
		fmt.Printf("Couldn't create sessions window: %v\n", err)
		return
	}
	ui.TagSet(w, "Load Refresh")
	ui.WindowDirty(w, false)

	list(w)

	for e := range w.EventChan() {
		switch e.C2 {
		case 'x', 'X': // execute
			text := string(e.Text)
			switch {
			case text == "Del":
				w.Ctl("delete")
				return
			case text == "Load":
				if len(e.Arg) > 0 {
					uuid := strings.TrimSpace(string(e.Arg))
					uuid = strings.Trim(uuid, `"'[]`) // Remove quotes and brackets
					if isUuid(uuid) {
						load(uuid, tracew)
						w.Ctl("delete")
						return
					} else {
						w.Fprintf("body", "\nInvalid UUID format: %s\n", uuid)
					}
				} else {
					w.Fprintf("body", "\nUsage: middle-click a UUID or 2-1 chord UUID into Load\n")
				}
			case text == "Refresh":
				list(w)
			case isUuid(text):
				load(text, tracew)
				w.Ctl("delete")
				return
			default:
				w.WriteEvent(e)
			}
		case 'l', 'L': // look
			w.Ctl("clean")
		}
	}
}

func load(uuid string, tracew *a.Win) {
	currentSession = uuid
	if tracew != nil {
		tracew.Fprintf("body", "Loaded session %s\n", uuid)
	}
}

func getSummary(path string) string {
	// Default fallback
	summary := "conversation"

	// Read first few lines to find summary
	file, err := os.Open(path)
	if err != nil {
		return summary
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer to handle large lines (up to 1MB)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		// Look for summary field in JSONL
		if strings.Contains(line, `"summary"`) {
			// Simple extraction - look for "summary":"..." pattern
			if idx := strings.Index(line, `"summary":"`); idx != -1 {
				start := idx + len(`"summary":"`)
				if end := strings.Index(line[start:], `"`); end != -1 {
					extracted := line[start : start+end]
					if extracted != "" {
						summary = extracted
						break
					}
				}
			}
		}
		// Also look for first user message as fallback
		if strings.Contains(line, `"role":"user"`) && strings.Contains(line, `"content"`) {
			if idx := strings.Index(line, `"content":"`); idx != -1 {
				start := idx + len(`"content":"`)
				if end := strings.Index(line[start:], `"`); end != -1 {
					content := line[start : start+end]
					if len(content) > 50 {
						content = content[:50] + "..."
					}
					if content != "" && summary == "conversation" {
						summary = content
					}
				}
			}
		}
	}
	// Check for scanner errors (e.g., lines too large for buffer)
	if err := scanner.Err(); err != nil {
		// Silently fall back to default summary on error
		summary = "conversation (parse error)"
	}

	return summary
}

func list(w *a.Win) {
	homeDir, _ := os.UserHomeDir()
	claudeProjectsDir := filepath.Join(homeDir, ".claude", "projects")
	cwd := util.Getwd()

	// Convert current directory to claude project path format
	currentDirPath := strings.ReplaceAll(cwd, "/", "-")
	projectDir := filepath.Join(claudeProjectsDir, currentDirPath)

	w.Clear()

	w.Fprintf("body", "# Claude Sessions for %s - highlight line and click Load\n\n", cwd)

	// Read session files
	files, err := os.ReadDir(projectDir)
	slices.SortFunc(files, func(a, b os.DirEntry) int {
		afi, _ := a.Info()
		bfi, _ := b.Info()

		return bfi.ModTime().Compare(afi.ModTime())
	})
	if err != nil {
		w.Fprintf("body", "No sessions found: %v\n", err)
		w.Ctl("clean")
		return
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".jsonl") {
			sessionID := strings.TrimSuffix(file.Name(), ".jsonl")
			summary := getSummary(filepath.Join(projectDir, file.Name()))
			w.Fprintf("body", "[%s] | %s\n", sessionID, summary)
		}
	}

	w.Ctl("clean")
}

func LastSessionId() string {
	homeDir, _ := os.UserHomeDir()
	claudeProjectsDir := filepath.Join(homeDir, ".claude", "projects")

	cwd := util.Getwd()

	// Convert current directory to claude project path format
	currentDirPath := strings.ReplaceAll(cwd, "/", "-")
	projectDir := filepath.Join(claudeProjectsDir, currentDirPath)

	// Read session files
	files, err := os.ReadDir(projectDir)
	if err != nil {
		return ""
	}

	// Sort by modification time (most recent first)
	slices.SortFunc(files, func(a, b os.DirEntry) int {
		afi, _ := a.Info()
		bfi, _ := b.Info()
		return bfi.ModTime().Compare(afi.ModTime())
	})

	// Find most recent .jsonl file
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".jsonl") {
			return strings.TrimSuffix(file.Name(), ".jsonl")
		}
	}

	return ""
}

func isUuid(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}
