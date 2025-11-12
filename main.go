package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"claude-acme/internal/acme"
	"claude-acme/internal/permissions"

	a "9fans.net/go/acme"
)

var (
	cwd              string
	pwname           string
	twname           string
	currentSessionID string
)

func main() {
	var err error

	cwd, err = os.Getwd()
	if err != nil {
		log.Fatalf("couldn't get working directory: %v", err)
	}

	pwname = filepath.Join(cwd, "+Claude")
	pw, err := acme.WindowOpen(pwname)
	if err != nil {
		log.Fatal(err)
	}
	defer pw.CloseFiles()
	if err = acme.TagSet(pwname, "Send Permissions Sessions"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set prompt window tag: %v\n", err)
		return
	}
	pw.Fprintf("body", "USER: [Send]\n")

	twname = filepath.Join(cwd, "+ClaudeTrace")
	tw, err := acme.WindowOpen(twname)
	if err != nil {
		tw.Fprintf("body", "Warning: Could not open trace window: %v\n", err)
		tw = nil
	}
	defer func() {
		if tw != nil {
			tw.CloseFiles()
		}
	}()

	for e := range pw.EventChan() {
		switch e.C2 {
		case 'x', 'X': // execute (middle-click)
			switch string(e.Text) {
			case "Send":
				executePrompt(pw, tw)
			case "Permissions":
				go permissionsWindow()
			case "Sessions":
				go sessionsWindow(tw)
			default:
				pw.WriteEvent(e)
			}
		case 'l', 'L': // look
			pw.WriteEvent(e)
		}
	}
}

func executePrompt(pw *a.Win, tw *a.Win) {
	// Read content from prompt window
	promptContent, err := acme.BodyRead(pwname)
	if err != nil {
		pw.Fprintf("body", "Error reading from prompt window: %v\n", err)
		return
	}

	promptContent = bytes.TrimSpace(promptContent)
	if len(promptContent) == 0 {
		acme.BodyWrite(pwname, "$", []byte("Prompt window is empty. Please enter your request in +Prompt window first.\n"))
		return
	}

	// Clear prompt window
	err = acme.BodyWrite(pwname, ",", []byte(""))
	if err != nil {
		pw.Fprintf("body", "Error clearing prompt window: %v\n", err)
		return
	}

	// Strip USER: prefix if present and display user input
	userInput := strings.TrimPrefix(string(promptContent), "USER: [Send]\n")
	pw.Fprintf("body", "\nUSER: [Send]\n%s\n\nCLAUDE:\n", userInput)

	// Load settings for tool permissions
	settings, err := permissions.Read(cwd)
	if err != nil {
		pw.Fprintf("body", "Error loading settings: %v\n", err)
		return
	}

	// Build claude command with arguments
	args := []string{"-p", "-d"}

	// Add session management
	if currentSessionID != "" {
		args = append(args, "-r", currentSessionID)
	} else {
		args = append(args, "-c")
	}

	if len(settings.AllowedTools) > 0 {
		args = append(args, "--allowedTools")
		args = append(args, strings.Join(settings.AllowedTools, ","))
	}

	if len(settings.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools")
		args = append(args, strings.Join(settings.DisallowedTools, ","))
	}

	if settings.PermissionMode != "" {
		args = append(args, "--permission-mode", settings.PermissionMode)
	}

	// Execute claude command
	cmd := exec.Command("claude", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		pw.Fprintf("body", "Error creating stdin pipe: %v\n", err)
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		pw.Fprintf("body", "Error creating stdout pipe: %v\n", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		pw.Fprintf("body", "Error creating stderr pipe: %v\n", err)
		return
	}

	err = cmd.Start()
	if err != nil {
		pw.Fprintf("body", "Error starting claude command: %v\n", err)
		return
	}

	// Send user input to claude (it will continue the latest session)
	go func() {
		defer stdin.Close()
		stdin.Write([]byte(userInput))
	}()

	// Handle stdout and stderr streams
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		handleClaudeOutput(pw, stdout, tw)
	}()
	go func() {
		defer wg.Done()
		handleClaudeOutput(pw, stderr, tw)
	}()

	// Wait for command and streams to finish
	err = cmd.Wait()
	wg.Wait()
	if err != nil {
		pw.Fprintf("body", "\n[Error: %v]", err)
		return
	}

	// Add separator
	pw.Fprintf("body", "\n====================\n\nUSER: [Send]\n")
}

func handleClaudeOutput(claudeWin *a.Win, stream io.Reader, traceWin *a.Win) {
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "[DEBUG] ") {
			// Send debug messages to trace window
			if traceWin != nil {
				traceWin.Fprintf("body", "%s\n", line)
			}
		} else {
			// Send regular output to claude window
			claudeWin.Fprintf("body", "%s\n", line)
		}
	}
}

func sessionsWindow(traceWin *a.Win) {
	sessWindow, err := a.New()
	if err != nil {
		fmt.Printf("Couldn't create sessions window: %v\n", err)
		return
	}

	sessWindowName := prependCwd("+Claude-Sessions")
	sessWindow.Name(sessWindowName)
	sessWindow.Fprintf("tag", "Load Refresh")
	sessWindow.Ctl("clean")

	// Show available sessions
	listSessions(sessWindow)

	// Event loop for sessions window
	for e := range sessWindow.EventChan() {
		switch e.C2 {
		case 'x', 'X': // execute
			text := string(e.Text)
			switch {
			case text == "Del":
				sessWindow.Ctl("delete")
				return
			case text == "Load":
				// Check if there's a chorded argument
				if len(e.Arg) > 0 {
					uuid := strings.TrimSpace(string(e.Arg))
					uuid = strings.Trim(uuid, `"'[]`) // Remove quotes and brackets
					if isValidUUID(uuid) {
						loadSession(uuid, traceWin)
						sessWindow.Ctl("delete")
						return
					} else {
						sessWindow.Fprintf("body", "\nInvalid UUID format: %s\n", uuid)
					}
				} else {
					sessWindow.Fprintf("body", "\nUsage: middle-click a UUID or 2-1 chord UUID into Load\n")
				}
			case text == "Refresh":
				listSessions(sessWindow)
			case isValidUUID(text):
				// Middle-clicked UUID
				loadSession(text, traceWin)
				sessWindow.Ctl("delete")
				return
			default:
				sessWindow.WriteEvent(e)
			}
		case 'l', 'L': // look
			sessWindow.Ctl("clean")
		}
	}
}

func permissionsWindow() {
	permWindow, err := a.New()
	if err != nil {
		fmt.Printf("Couldn't create permissions window: %v\n", err)
		return
	}

	permWindowName := prependCwd("+Claude-Permissions")
	permWindow.Name(permWindowName)
	permWindow.Fprintf("tag", "Show Edit Save")
	permWindow.Ctl("clean")

	// Show current permissions
	showCurrentPermissions(permWindow)

	// Event loop for permissions window
	for e := range permWindow.EventChan() {
		switch e.C2 {
		case 'x', 'X': // execute
			switch string(e.Text) {
			case "Del":
				permWindow.Ctl("delete")
				return
			case "Show":
				showCurrentPermissions(permWindow)
			case "Edit":
				listAllToolsForEditing(permWindow)
			case "Save":
				savePermissionChanges(permWindow)
			default:
				permWindow.WriteEvent(e)
			}
		case 'l', 'L': // look
			permWindow.Ctl("clean")
		}
	}
}

func showCurrentPermissions(w *a.Win) {
	perms, err := permissions.Read(cwd)
	if err != nil {
		w.Fprintf("body", "Error loading permissions: %v\n", err)
		return
	}

	w.Clear()
	w.Fprintf("body", "# Active permissions for: %s\n", cwd)
	w.Fprintf("body", "\n")

	for _, tool := range perms.AllowedTools {
		w.Fprintf("body", "+ %s\n", tool)
	}

	for _, tool := range perms.DisallowedTools {
		w.Fprintf("body", "- %s\n", tool)
	}

	w.Ctl("clean")
}

var allAvailableTools = []string{
	"Read", "Write", "Edit", "MultiEdit", "NotebookEdit",
	"Glob", "Grep", "Bash", "BashOutput", "KillBash",
	"WebSearch", "WebFetch", "Task", "TodoWrite", "ExitPlanMode",
	"Bash(git:*)", "Bash(mkdir:*)", "Bash(ls:*)", "Bash(cd:*)",
	"Bash(cp:*)", "Bash(mv:*)", "Bash(rm:*)", "Bash(chmod:*)",
	"Edit(/path/to/dir/*)", "Read(/path/to/dir/*)", "Write(/path/to/dir/*)",
}

func listAllToolsForEditing(w *a.Win) {
	perms, err := permissions.Read(cwd)
	if err != nil {
		w.Fprintf("body", "Error loading permissions: %v\n", err)
		return
	}

	allowedSet := make(map[string]bool)
	for _, tool := range perms.AllowedTools {
		allowedSet[tool] = true
	}

	w.Clear()
	w.Fprintf("body", "# Available tools to grant - edit with + to allow, - to deny, ~ to remove\n\n")

	for _, tool := range allAvailableTools {
		if !allowedSet[tool] {
			w.Fprintf("body", "  %s\n", tool)
		}
	}

	w.Ctl("clean")
}

func savePermissionChanges(w *a.Win) {
	content, err := w.ReadAll("body")
	if err != nil {
		w.Fprintf("body", "Error reading window content: %v\n", err)
		return
	}

	perms, err := permissions.Read(cwd)
	if err != nil {
		w.Fprintf("body", "Error loading settings: %v\n", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	var toAllow, toDeny, toRemove []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "+") {
			tool := strings.TrimSpace(line[1:])
			if tool != "" {
				toAllow = append(toAllow, tool)
			}
		} else if strings.HasPrefix(line, "-") {
			tool := strings.TrimSpace(line[1:])
			if tool != "" {
				toDeny = append(toDeny, tool)
			}
		} else if strings.HasPrefix(line, "~") {
			tool := strings.TrimSpace(line[1:])
			if tool != "" {
				toRemove = append(toRemove, tool)
			}
		}
	}

	// Update settings
	for _, tool := range toAllow {
		if !slices.Contains(perms.AllowedTools, tool) {
			perms.AllowedTools = append(perms.AllowedTools, tool)
		}
		perms.DisallowedTools = remove(perms.DisallowedTools, tool)
	}

	for _, tool := range toDeny {
		if !slices.Contains(perms.DisallowedTools, tool) {
			perms.DisallowedTools = append(perms.DisallowedTools, tool)
		}
		perms.AllowedTools = remove(perms.AllowedTools, tool)
	}

	for _, tool := range toRemove {
		perms.AllowedTools = remove(perms.AllowedTools, tool)
		perms.DisallowedTools = remove(perms.DisallowedTools, tool)
	}

	err = permissions.Write(cwd, perms)
	if err != nil {
		w.Fprintf("body", "Error saving permissions: %v\n", err)
		return
	}

	w.Fprintf("body", "\n✓ Permissions updated successfully!\n")
	showCurrentPermissions(w)
}

func prependCwd(suffix string) string {
	pwd, err := os.Getwd()
	if err != nil {
		pwd = ""
	}
	return filepath.Join(pwd, suffix)
}

func remove(slice []string, item string) []string {
	result := make([]string, 0)
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

func listSessions(w *a.Win) {
	homeDir, _ := os.UserHomeDir()
	claudeProjectsDir := filepath.Join(homeDir, ".claude", "projects")

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
			summary := getSessionSummary(filepath.Join(projectDir, file.Name()))
			w.Fprintf("body", "[%s] | %s\n", sessionID, summary)
		}
	}

	w.Ctl("clean")
}

func getSessionSummary(filePath string) string {
	// Default fallback
	summary := "conversation"

	// Read first few lines to find summary
	file, err := os.Open(filePath)
	if err != nil {
		return summary
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
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

	return summary
}

func isValidUUID(s string) bool {
	// UUID pattern: 8-4-4-4-12 hex characters
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

func loadSession(uuid string, traceWin *a.Win) {
	currentSessionID = uuid
	if traceWin != nil {
		traceWin.Fprintf("body", "Loaded session %s\n", uuid)
	}
}
