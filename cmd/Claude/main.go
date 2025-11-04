package main

import (
	"log"

	"claude-acme/internal/acme"
)

func main() {
	client := acme.NewClient()
	defer client.Close()

	promptWindowName := acme.GetWindowName("+Prompt")
	claudeWindowName := acme.GetWindowName("+Claude")

	_, err := client.CreateWindow(promptWindowName)
	if err != nil {
		log.Fatalf("Failed to create prompt window: %v", err)
	}

	_, err = client.CreateWindow(claudeWindowName)
	if err != nil {
		log.Fatalf("Failed to create Claude window: %v", err)
	}

	err = client.SetWindowTag(promptWindowName, " Prompt")
	if err != nil {
		log.Fatalf("Failed to set prompt window tag: %v", err)
	}

}