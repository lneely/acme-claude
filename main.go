package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"claude-acme/internal/debug"
	"claude-acme/internal/permissions"
	"claude-acme/internal/sessions"
	"claude-acme/internal/ui"
	"claude-acme/internal/util"

	a "9fans.net/go/acme"
)

func main() {
	var pw, tw *a.Win
	var err error

	cwd := util.Getwd()

	if pw, err = ui.WindowOpen(filepath.Join(cwd, "+Claude")); err != nil {
		log.Fatal(err)
	}
	defer pw.CloseFiles()

	if err = ui.TagSet(pw, "Send Permissions Sessions"); err != nil {
		log.Fatal(err)
	}
	pw.Fprintf("body", "USER: [Send]\n")

	if tw, err = ui.WindowOpen(filepath.Join(cwd, "+ClaudeTrace")); err != nil {
		log.Printf("failed to create trace window: %v", err)
	}
	defer func() {
		if tw != nil {
			tw.CloseFiles()
		}
	}()

	for e := range pw.EventChan() {
		if e.C2 == 'x' || e.C2 == 'X' {
			switch string(e.Text) {
			case "Send":
				sendPrompt(pw, tw)
			case "Permissions":
				go permissions.Run()
			case "Sessions":
				go sessions.Run(tw)
			default:
				pw.WriteEvent(e)
			}
		} else {
			pw.WriteEvent(e)
		}
	}
}

func handleClaudeOutput(claudeWin *a.Win, stream io.Reader, traceWin *a.Win) {
	scanner := bufio.NewScanner(stream)
	// Increase buffer to handle large lines (up to 1MB)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
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
	// Check for scanner errors (e.g., lines too large for buffer)
	if err := scanner.Err(); err != nil {
		claudeWin.Fprintf("body", "\n[Scanner Error: %v]\n", err)
	}
}

func sendPrompt(pw *a.Win, tw *a.Win) {
	// Read content from prompt window
	promptContent, err := ui.BodyRead(pw)
	if err != nil {
		pw.Fprintf("body", "Error reading from prompt window: %v\n", err)
		return
	}

	promptContent = bytes.TrimSpace(promptContent)
	if len(promptContent) == 0 {
		ui.BodyWrite(pw, "$", []byte("Prompt window is empty. Please enter your request in +Prompt window first.\n"))
		return
	}

	// Clear prompt window
	err = ui.BodyWrite(pw, ",", []byte(""))
	if err != nil {
		pw.Fprintf("body", "Error clearing prompt window: %v\n", err)
		return
	}

	// Strip USER: prefix if present and display user input
	userInput := strings.TrimPrefix(string(promptContent), "USER: [Send]\n")
	pw.Fprintf("body", "\nUSER: [Send]\n%s\n\nCLAUDE:\n", userInput)

	// Load settings for tool permissions
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("couldn't get working directory: %v", err)
	}
	perms, err := permissions.Read(cwd)
	if err != nil {
		pw.Fprintf("body", "Error loading settings: %v\n", err)
		return
	}

	// Build claude command with arguments
	args := []string{"-p", "-d"}

	// Add session management
	var sessionID string
	if sessions.CurrentSessionId() != "" {
		args = append(args, "-r", sessions.CurrentSessionId())
		sessionID = sessions.CurrentSessionId()
	} else {
		// Get the most recent session ID and resume it explicitly
		sessionID = sessions.LastSessionId()
		if sessionID != "" {
			args = append(args, "-r", sessionID)
		} else {
			// No existing session, create new one
			args = append(args, "-c")
		}
	}

	allowed := perms.GetAllowed()
	if len(allowed) > 0 {
		args = append(args, "--allowedTools")
		args = append(args, "\""+strings.Join(allowed, ",")+"\"")
	}

	disallowed := perms.GetDisallowed()
	if len(disallowed) > 0 {
		args = append(args, "--disallowedTools")
		args = append(args, "\""+strings.Join(disallowed, ",")+"\"")
	}

	permMode := perms.PermissionMode
	if permMode == "" {
		permMode = "default"
	}
	args = append(args, "--permission-mode", permMode)

	if tw != nil {
		tw.Fprintf("body", "Executing claude with args: %v\n", args)
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

	// Start tailing debug logs for all sessions if trace window exists
	ctx, cancel := context.WithCancel(context.Background())
	if tw != nil {
		tw.Fprintf("body", "[TRACE] Starting debug log monitoring\n")
		wg.Add(1)
		go func() {
			defer wg.Done()
			debug.Tail(ctx, tw)
		}()
	}

	// Wait for command and streams to finish
	// IMPORTANT: Must wait for goroutines to finish reading BEFORE cmd.Wait()
	// Per Go docs: "It is incorrect to call Wait before all reads from the pipe have completed"
	wg.Wait()
	cancel()
	err = cmd.Wait()
	if err != nil {
		pw.Fprintf("body", "\n[Error: %v]", err)
		return
	}

	// Add separator
	pw.Fprintf("body", "\n====================\n\nUSER: [Send]\n")
}
