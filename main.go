package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"claude-acme/internal/acme"
	"claude-acme/internal/context"

	a "9fans.net/go/acme"
)

var (
	ctx   *context.Manager
	cwd   string
	wname string
)

func main() {
	var err error

	ctx, err = context.NewManager()
	if err != nil {
		log.Fatalf("Failed to create context manager: %v", err)
	}

	wname = prependCwd("+Claude")

	pw, err := acme.WindowOpen(wname)
	if err != nil {
		log.Fatal(err)
	}
	defer pw.CloseFiles()
	if err = acme.TagSet(wname, "Send Permissions Reset"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set prompt window tag: %v\n", err)
		return
	}

	for e := range pw.EventChan() {
		switch e.C2 {
		case 'x', 'X': // execute (middle-click)
			switch string(e.Text) {
			case "Send":
				executePrompt(pw)
			case "Reset":
				executeReset(pw)
			case "Permissions":
				executePermissions()
			default:
				pw.WriteEvent(e)
			}
		case 'l', 'L': // look
			pw.Ctl("clean")
		}
	}
}

func executePrompt(w *a.Win) {
	// Read content from prompt window
	promptContent, err := acme.BodyRead(wname)
	if err != nil {
		w.Fprintf("body", "Error reading from prompt window: %v\n", err)
		return
	}

	promptContent = bytes.TrimSpace(promptContent)
	if len(promptContent) == 0 {
		acme.BodyWrite(wname, "$", []byte("Prompt window is empty. Please enter your request in +Prompt window first.\n"))
		return
	}

	// Clear prompt window
	err = acme.BodyWrite(wname, ",", []byte(""))
	if err != nil {
		w.Fprintf("body", "Error clearing prompt window: %v\n", err)
		return
	}

	// Display user input
	w.Fprintf("body", "\nUSER:\n%s\n\nCLAUDE:\n", promptContent)

	// Build full prompt with context
	fullPrompt, err := ctx.BuildPrompt(cwd, string(promptContent))
	if err != nil {
		w.Fprintf("body", "Error building prompt with context: %v\n", err)
		return
	}

	// Load settings for tool permissions
	settings, err := ctx.LoadSettings(cwd)
	if err != nil {
		w.Fprintf("body", "Error loading settings: %v\n", err)
		return
	}

	// Build claude command with arguments
	args := []string{"-p", "--debug"}

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
		w.Fprintf("body", "Error creating stdin pipe: %v\n", err)
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		w.Fprintf("body", "Error creating stdout pipe: %v\n", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		w.Fprintf("body", "Error creating stderr pipe: %v\n", err)
		return
	}

	err = cmd.Start()
	if err != nil {
		w.Fprintf("body", "Error starting claude command: %v\n", err)
		return
	}

	// Send prompt to claude
	go func() {
		defer stdin.Close()
		stdin.Write([]byte(fullPrompt))
	}()

	// Process output
	var claudeResponse strings.Builder
	var errorOutput strings.Builder

	// Read stderr for debug output - route ALL stderr to os.Stderr
	go func() {
		stderrScanner := bufio.NewScanner(stderr)
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			if actionMsg, isDebug := parseDebugMessage(line); isDebug {
				fmt.Fprintln(os.Stderr, actionMsg)
			} else {
				// Route all other stderr content to stderr as well
				fmt.Fprintln(os.Stderr, line)
				errorOutput.WriteString(line + "\n")
			}
		}
	}()

	// Read and stream stdout to window
	stdoutScanner := bufio.NewScanner(stdout)
	for stdoutScanner.Scan() {
		line := stdoutScanner.Text()

		if !strings.HasPrefix(line, "[DEBUG]") && strings.TrimSpace(line) != "" {
			claudeResponse.WriteString(line + "\n")
			w.Fprintf("body", "%s\n", line)
		}
	}

	// Wait for command to finish
	err = cmd.Wait()
	if err != nil {
		errorMsg := fmt.Sprintf("\n[Error: %v]", err)
		if errorOutput.Len() > 0 {
			errorMsg += "\nClaude CLI Error Output:\n" + errorOutput.String()
		}
		w.Fprintf("body", errorMsg)
		return
	}

	// Add separator
	w.Fprintf("body", "\n====================\n\nUSER:\n")

	// Save to context
	err = ctx.AddMessage(cwd, "user", string(promptContent))
	if err != nil {
		w.Fprintf("body", "Warning: Failed to save user message to context: %v\n", err)
	}

	err = ctx.AddMessage(cwd, "assistant", strings.TrimSpace(claudeResponse.String()))
	if err != nil {
		w.Fprintf("body", "Warning: Failed to save Claude response to context: %v\n", err)
	}
}

func executeReset(w *a.Win) {
	err := ctx.ClearContext(cwd)
	if err != nil {
		w.Fprintf("body", "Error clearing context: %v\n", err)
		return
	}

	w.Fprintf("body", "✓ Cleared Claude context for directory: %s\n", cwd)
}

func executePermissions() {
	// Create permissions window similar to how Ampd creates sub-windows
	go permissionsWindow()
}

func parseDebugMessage(line string) (string, bool) {
	if strings.HasPrefix(line, "[DEBUG]") {
		cleanLine := strings.TrimPrefix(line, "[DEBUG] ")
		return "🔍 " + cleanLine, true
	}
	return "", false
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
	settings, err := ctx.LoadSettings(cwd)
	if err != nil {
		w.Fprintf("body", "Error loading settings: %v\n", err)
		return
	}

	w.Clear()
	w.Fprintf("body", "# Active permissions for: %s\n", cwd)
	w.Fprintf("body", "# Permission Mode: %s\n", settings.PermissionMode)
	if len(settings.AdditionalDirs) > 0 {
		w.Fprintf("body", "# Additional Directories: %s\n", strings.Join(settings.AdditionalDirs, ", "))
	}
	w.Fprintf("body", "\n")

	for _, tool := range settings.AllowedTools {
		w.Fprintf("body", "+ %s\n", tool)
	}

	for _, tool := range settings.DisallowedTools {
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
	settings, err := ctx.LoadSettings(cwd)
	if err != nil {
		w.Fprintf("body", "Error loading settings: %v\n", err)
		return
	}

	allowedSet := make(map[string]bool)
	for _, tool := range settings.AllowedTools {
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

	settings, err := ctx.LoadSettings(cwd)
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
		if !slices.Contains(settings.AllowedTools, tool) {
			settings.AllowedTools = append(settings.AllowedTools, tool)
		}
		settings.DisallowedTools = remove(settings.DisallowedTools, tool)
	}

	for _, tool := range toDeny {
		if !slices.Contains(settings.DisallowedTools, tool) {
			settings.DisallowedTools = append(settings.DisallowedTools, tool)
		}
		settings.AllowedTools = remove(settings.AllowedTools, tool)
	}

	for _, tool := range toRemove {
		settings.AllowedTools = remove(settings.AllowedTools, tool)
		settings.DisallowedTools = remove(settings.DisallowedTools, tool)
	}

	err = ctx.SaveSettings(cwd, settings)
	if err != nil {
		w.Fprintf("body", "Error saving settings: %v\n", err)
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
