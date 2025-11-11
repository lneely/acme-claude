package context

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type Context struct {
	Messages []Message `json:"messages"`
	LastUsed time.Time `json:"last_used"`
}

func getContextDir(cwd string) string {
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".claude-acme")
	
	hash := sha256.Sum256([]byte(cwd))
	dirHash := hex.EncodeToString(hash[:])
	contextDir := filepath.Join(baseDir, dirHash)
	os.MkdirAll(contextDir, 0755)
	return contextDir
}

func getContextPath(cwd string) string {
	return filepath.Join(getContextDir(cwd), "context.json")
}

func LoadContext(cwd string) (*Context, error) {
	contextPath := getContextPath(cwd)
	
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		return &Context{
			Messages: []Message{},
			LastUsed: time.Now(),
		}, nil
	}

	data, err := os.ReadFile(contextPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read context file: %w", err)
	}

	var ctx Context
	err = json.Unmarshal(data, &ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse context file: %w", err)
	}

	return &ctx, nil
}

func SaveContext(cwd string, ctx *Context) error {
	contextPath := getContextPath(cwd)
	ctx.LastUsed = time.Now()

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	err = os.WriteFile(contextPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}

	return nil
}

func BuildPrompt(cwd string, newPrompt string) (string, error) {
	ctx, err := LoadContext(cwd)
	if err != nil {
		return "", err
	}

	var fullPrompt string
	
	for _, msg := range ctx.Messages {
		switch msg.Role {
		case "user":
			fullPrompt += "USER: " + msg.Content + "\n\n"
		case "assistant":
			fullPrompt += "CLAUDE: " + msg.Content + "\n\n"
		}
	}
	
	if len(ctx.Messages) > 0 {
		fullPrompt += "====================\n\n"
	}
	
	fullPrompt += "USER: " + newPrompt

	return fullPrompt, nil
}