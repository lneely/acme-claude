package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"9fans.net/go/acme"
	internalAcme "claude-acme/internal/acme"
	"claude-acme/internal/context"
)

func main() {
	client := internalAcme.NewClient()
	defer client.Close()

	contextManager, err := context.NewManager()
	if err != nil {
		log.Fatalf("Failed to create context manager: %v", err)
	}

	workingDir := internalAcme.GetCurrentWorkingDir()
	if workingDir == "" {
		log.Fatalf("Failed to get current working directory")
	}

	permissionsWindowName := internalAcme.GetWindowName("+Claude-Permissions")

	// Check if window already exists in acme
	windowExists := false
	if wins, err := acme.Windows(); err == nil {
		for _, info := range wins {
			if info.Name == permissionsWindowName {
				windowExists = true
				break
			}
		}
	}

	_, err = client.OpenWindow(permissionsWindowName)
	if err != nil {
		log.Fatalf("Failed to open permissions window: %v", err)
	}

	// Only set tag if this is a new window
	if !windowExists {
		err = client.SetWindowTag(permissionsWindowName, " Claude-Permissions")
		if err != nil {
			log.Fatalf("Failed to set permissions window tag: %v", err)
		}
	}

	err = client.ClearWindow(permissionsWindowName)
	if err != nil {
		log.Fatalf("Failed to clear permissions window: %v", err)
	}

	if len(os.Args) < 2 {
		showCurrentPermissions(client, contextManager, workingDir, permissionsWindowName)
		return
	}

	arg := os.Args[1]

	if arg == "?" {
		listAllToolsForEditing(client, contextManager, workingDir, permissionsWindowName)
		return
	}

	parsePermissionChanges(client, contextManager, workingDir, permissionsWindowName, arg)
}

var allAvailableTools = []string{
	// File Operations
	"Read",
	"Write", 
	"Edit",
	"MultiEdit",
	"NotebookEdit",
	
	// Search & Discovery
	"Glob",
	"Grep",
	
	// System Operations
	"Bash",
	"BashOutput",
	"KillBash",
	
	// Web & Network
	"WebSearch",
	"WebFetch",
	
	// Task Management
	"Task",
	"TodoWrite",
	"ExitPlanMode",
	
	// Special patterns (examples)
	"Bash(git:*)",
	"Bash(mkdir:*)",
	"Bash(ls:*)",
	"Bash(cd:*)",
	"Bash(cp:*)",
	"Bash(mv:*)",
	"Bash(rm:*)",
	"Bash(chmod:*)",
	"Bash(find:*)",
	"Bash(grep:*)",
	"Bash(awk:*)",
	"Bash(sed:*)",
	"Bash(sort:*)",
	"Bash(uniq:*)",
	"Bash(head:*)",
	"Bash(tail:*)",
	"Bash(cat:*)",
	"Bash(less:*)",
	"Bash(more:*)",
	"Bash(which:*)",
	"Bash(whereis:*)",
	"Bash(file:*)",
	"Bash(stat:*)",
	"Bash(du:*)",
	"Bash(df:*)",
	"Bash(ps:*)",
	"Bash(top:*)",
	"Bash(kill:*)",
	"Bash(killall:*)",
	"Bash(jobs:*)",
	"Bash(bg:*)",
	"Bash(fg:*)",
	"Bash(nohup:*)",
	"Bash(screen:*)",
	"Bash(tmux:*)",
	"Bash(ssh:*)",
	"Bash(scp:*)",
	"Bash(rsync:*)",
	"Bash(curl:*)",
	"Bash(wget:*)",
	"Bash(tar:*)",
	"Bash(zip:*)",
	"Bash(unzip:*)",
	"Bash(gzip:*)",
	"Bash(gunzip:*)",
	"Edit(/path/to/dir/*)",
	"Read(/path/to/dir/*)",
	"Write(/path/to/dir/*)",
}

func showCurrentPermissions(client *internalAcme.Client, manager *context.Manager, workingDir string, windowName string) {
	settings, err := manager.LoadSettings(workingDir)
	if err != nil {
		log.Fatalf("Failed to load settings: %v", err)
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Active permissions for: %s\n", workingDir))
	output.WriteString(fmt.Sprintf("# Permission Mode: %s\n", settings.PermissionMode))
	if len(settings.AdditionalDirs) > 0 {
		output.WriteString(fmt.Sprintf("# Additional Directories: %s\n", strings.Join(settings.AdditionalDirs, ", ")))
	}
	output.WriteString("\n")

	// Show only explicitly allowed tools
	for _, tool := range settings.AllowedTools {
		output.WriteString(fmt.Sprintf("+ %s\n", tool))
	}

	// Show only explicitly denied tools
	for _, tool := range settings.DisallowedTools {
		output.WriteString(fmt.Sprintf("- %s\n", tool))
	}

	err = client.WriteToWindow(windowName, output.String())
	if err != nil {
		log.Fatalf("Failed to write to permissions window: %v", err)
	}
}

func listAllToolsForEditing(client *internalAcme.Client, manager *context.Manager, workingDir string, windowName string) {
	settings, err := manager.LoadSettings(workingDir)
	if err != nil {
		log.Fatalf("Failed to load settings: %v", err)
	}

	allowedSet := make(map[string]bool)
	for _, tool := range settings.AllowedTools {
		allowedSet[tool] = true
	}

	var output strings.Builder
	output.WriteString("# Available tools to grant - edit with + to allow, - to deny, ~ to remove\n")
	output.WriteString("\n")
	
	for _, tool := range allAvailableTools {
		if !allowedSet[tool] {
			output.WriteString(fmt.Sprintf("  %s\n", tool))
		}
	}

	err = client.WriteToWindow(windowName, output.String())
	if err != nil {
		log.Fatalf("Failed to write to permissions window: %v", err)
	}
}

func parsePermissionChanges(client *internalAcme.Client, manager *context.Manager, workingDir string, windowName string, input string) {
	settings, err := manager.LoadSettings(workingDir)
	if err != nil {
		log.Fatalf("Failed to load settings: %v", err)
	}

	lines := strings.Split(input, "\n")
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

	for _, tool := range toAllow {
		if !contains(settings.AllowedTools, tool) {
			settings.AllowedTools = append(settings.AllowedTools, tool)
		}
		settings.DisallowedTools = remove(settings.DisallowedTools, tool)
	}

	for _, tool := range toDeny {
		if !contains(settings.DisallowedTools, tool) {
			settings.DisallowedTools = append(settings.DisallowedTools, tool)
		}
		settings.AllowedTools = remove(settings.AllowedTools, tool)
	}

	for _, tool := range toRemove {
		settings.AllowedTools = remove(settings.AllowedTools, tool)
		settings.DisallowedTools = remove(settings.DisallowedTools, tool)
	}

	err = manager.SaveSettings(workingDir, settings)
	if err != nil {
		log.Fatalf("Failed to save settings: %v", err)
	}

	var output strings.Builder
	if len(toAllow) > 0 {
		output.WriteString(fmt.Sprintf("Allowed: %s\n", strings.Join(toAllow, ", ")))
	}
	if len(toDeny) > 0 {
		output.WriteString(fmt.Sprintf("Denied: %s\n", strings.Join(toDeny, ", ")))
	}
	if len(toRemove) > 0 {
		output.WriteString(fmt.Sprintf("Removed: %s\n", strings.Join(toRemove, ", ")))
	}
	
	if output.Len() > 0 {
		err = client.WriteToWindow(windowName, output.String())
		if err != nil {
			log.Fatalf("Failed to write to permissions window: %v", err)
		}
	}

	// Show updated permissions
	showCurrentPermissions(client, manager, workingDir, windowName)
}


func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
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