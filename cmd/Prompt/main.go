package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"claude-acme/internal/acme"
	"claude-acme/internal/context"
)

// parseDebugMessage extracts action information from debug messages
func parseDebugMessage(line string) (string, bool) {
	// Show ALL debug messages that start with [DEBUG] - full transparency
	if strings.HasPrefix(line, "[DEBUG]") {
		// Remove the [DEBUG] prefix and add our own formatting
		cleanLine := strings.TrimPrefix(line, "[DEBUG] ")
		return "🔍 " + cleanLine, true
	}

	return "", false
}

func main() {
	client := acme.NewClient()
	defer client.Close()

	contextManager, err := context.NewManager()
	if err != nil {
		log.Fatalf("Failed to create context manager: %v", err)
	}

	workingDir := acme.GetCurrentWorkingDir()
	if workingDir == "" {
		log.Fatalf("Failed to get current working directory")
	}

	promptWindowName := acme.GetWindowName("+Prompt")
	claudeWindowName := acme.GetWindowName("+Claude")

	promptContent, err := client.ReadFromWindow(promptWindowName)
	if err != nil {
		log.Fatalf("Failed to read from prompt window: %v", err)
	}

	promptContent = strings.TrimSpace(promptContent)
	if promptContent == "" {
		return
	}

	err = client.ClearWindow(promptWindowName)
	if err != nil {
		log.Fatalf("Failed to clear prompt window: %v", err)
	}

	userText := "USER:\n" + promptContent + "\n\n"
	err = client.AppendToWindow(claudeWindowName, userText)
	if err != nil {
		log.Fatalf("Failed to write user text to Claude window: %v", err)
	}

	claudeHeader := "CLAUDE:\n"
	err = client.AppendToWindow(claudeWindowName, claudeHeader)
	if err != nil {
		log.Fatalf("Failed to write Claude header to window: %v", err)
	}

	fullPrompt, err := contextManager.BuildPrompt(workingDir, promptContent)
	if err != nil {
		log.Fatalf("Failed to build prompt with context: %v", err)
	}

	settings, err := contextManager.LoadSettings(workingDir)
	if err != nil {
		log.Fatalf("Failed to load settings: %v", err)
	}

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

	cmd := exec.Command("claude", args...)

	// Set up stdin to provide the prompt
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to create stderr pipe: %v", err)
	}

	err = cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start claude command: %v", err)
	}

	// Write the prompt to stdin and close it
	go func() {
		defer stdin.Close()
		stdin.Write([]byte(fullPrompt))
	}()

	// Create scanners for both stdout and stderr
	stdoutScanner := bufio.NewScanner(stdout)
	stderrScanner := bufio.NewScanner(stderr)
	var claudeResponse strings.Builder
	var errorOutput strings.Builder

	// Read stderr in a goroutine
	go func() {
		for stderrScanner.Scan() {
			errorOutput.WriteString(stderrScanner.Text() + "\n")
		}
	}()

	for stdoutScanner.Scan() {
		line := stdoutScanner.Text()

		// Check if this is a debug message
		if actionMsg, isDebug := parseDebugMessage(line); isDebug {
			// Print debug output to stderr instead of Claude window
			fmt.Fprintln(os.Stderr, actionMsg)
			continue
		}

		// Check if this line starts the actual response (not debug)
		if !strings.HasPrefix(line, "[DEBUG]") && strings.TrimSpace(line) != "" {
			// This is part of Claude's actual response
			claudeResponse.WriteString(line + "\n")
			err = client.AppendToWindow(claudeWindowName, line+"\n")
			if err != nil {
				log.Fatalf("Failed to stream output to window: %v", err)
			}
		}
	}

	err = cmd.Wait()
	if err != nil {
		errorMsg := "\n[Error: " + err.Error() + "]"
		if errorOutput.Len() > 0 {
			errorMsg += "\nClaude CLI Error Output:\n" + errorOutput.String()
		}
		err = client.AppendToWindow(claudeWindowName, errorMsg)
		if err != nil {
			log.Fatalf("Failed to write error to window: %v", err)
		}
		return
	}

	footer := "\n\n====================\n\n"
	err = client.AppendToWindow(claudeWindowName, footer)
	if err != nil {
		log.Fatalf("Failed to write footer to window: %v", err)
	}

	err = contextManager.AddMessage(workingDir, "user", promptContent)
	if err != nil {
		log.Fatalf("Failed to save user message to context: %v", err)
	}

	err = contextManager.AddMessage(workingDir, "assistant", strings.TrimSpace(claudeResponse.String()))
	if err != nil {
		log.Fatalf("Failed to save Claude response to context: %v", err)
	}
}