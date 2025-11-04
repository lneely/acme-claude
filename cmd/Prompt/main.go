package main

import (
	"bufio"
	"log"
	"os/exec"
	"strings"

	"claude-acme/internal/acme"
	"claude-acme/internal/context"
)

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

	args := []string{"-p", fullPrompt}
	
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
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create stdout pipe: %v", err)
	}

	err = cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start claude command: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	var claudeResponse strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		claudeResponse.WriteString(line + "\n")
		err = client.AppendToWindow(claudeWindowName, line+"\n")
		if err != nil {
			log.Fatalf("Failed to stream output to window: %v", err)
		}
	}

	err = cmd.Wait()
	if err != nil {
		errorMsg := "\n[Error: " + err.Error() + "]"
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