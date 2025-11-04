package main

import (
	"fmt"
	"log"

	"claude-acme/internal/acme"
	"claude-acme/internal/context"
)

func main() {
	contextManager, err := context.NewManager()
	if err != nil {
		log.Fatalf("Failed to create context manager: %v", err)
	}

	workingDir := acme.GetCurrentWorkingDir()
	if workingDir == "" {
		log.Fatalf("Failed to get current working directory")
	}

	err = contextManager.ClearContext(workingDir)
	if err != nil {
		log.Fatalf("Failed to clear context: %v", err)
	}

	fmt.Printf("Cleared Claude context for directory: %s\n", workingDir)
}