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
	cwd    string
	pwname string
	twname string
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
	if err = acme.TagSet(pwname, "Send Permissions Reset"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set prompt window tag: %v\n", err)
		return
	}

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
			case "Reset":
				executeReset(tw)
			case "Permissions":
				go permissionsWindow()
			default:
				pw.WriteEvent(e)
			}
		case 'l', 'L': // look
			pw.Ctl("clean")
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
	userInput := strings.TrimPrefix(string(promptContent), "USER:\n")
	pw.Fprintf("body", "\nUSER:\n%s\n\nCLAUDE:\n", userInput)

	// Load settings for tool permissions
	settings, err := permissions.Read(cwd)
	if err != nil {
		pw.Fprintf("body", "Error loading settings: %v\n", err)
		return
	}

	// Build claude command with arguments - use -c to continue latest session
	args := []string{"-p", "-d", "-c"}

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
	pw.Fprintf("body", "\n====================\n\nUSER:\n")
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

func executeReset(w *a.Win) {
	w.Fprintf("body", "Reset functionality removed - Claude manages sessions automatically with -c flag\n")
	w.Fprintf("body", "Each new conversation will continue the latest session.\n")
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
