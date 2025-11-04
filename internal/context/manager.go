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

type Manager struct {
	baseDir string
}

type Settings struct {
	AllowedTools     []string `json:"allowedTools,omitempty"`
	DisallowedTools  []string `json:"disallowedTools,omitempty"`
	PermissionMode   string   `json:"permissionMode,omitempty"`
	AdditionalDirs   []string `json:"additionalDirs,omitempty"`
}

func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".claude-acme")
	err = os.MkdirAll(baseDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create context directory: %w", err)
	}

	return &Manager{baseDir: baseDir}, nil
}

func (m *Manager) getContextPath(workingDir string) string {
	hash := sha256.Sum256([]byte(workingDir))
	dirHash := hex.EncodeToString(hash[:])
	contextDir := filepath.Join(m.baseDir, dirHash)
	os.MkdirAll(contextDir, 0755)
	return filepath.Join(contextDir, "context.json")
}

func (m *Manager) LoadContext(workingDir string) (*Context, error) {
	contextPath := m.getContextPath(workingDir)
	
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

func (m *Manager) SaveContext(workingDir string, ctx *Context) error {
	contextPath := m.getContextPath(workingDir)
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

func (m *Manager) AddMessage(workingDir string, role string, content string) error {
	ctx, err := m.LoadContext(workingDir)
	if err != nil {
		return err
	}

	message := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	ctx.Messages = append(ctx.Messages, message)
	return m.SaveContext(workingDir, ctx)
}

func (m *Manager) ClearContext(workingDir string) error {
	contextPath := m.getContextPath(workingDir)
	
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(contextPath)
}

func (m *Manager) getSettingsPath(workingDir string) string {
	hash := sha256.Sum256([]byte(workingDir))
	dirHash := hex.EncodeToString(hash[:])
	contextDir := filepath.Join(m.baseDir, dirHash)
	os.MkdirAll(contextDir, 0755)
	return filepath.Join(contextDir, "settings.json")
}

func (m *Manager) LoadSettings(workingDir string) (*Settings, error) {
	settingsPath := m.getSettingsPath(workingDir)
	
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return &Settings{
			AllowedTools:   []string{"Bash", "Read", "Write", "Edit"},
			PermissionMode: "acceptEdits",
		}, nil
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	var settings Settings
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse settings file: %w", err)
	}

	return &settings, nil
}

func (m *Manager) SaveSettings(workingDir string, settings *Settings) error {
	settingsPath := m.getSettingsPath(workingDir)

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	err = os.WriteFile(settingsPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

func (m *Manager) GetSettingsPath(workingDir string) string {
	return m.getSettingsPath(workingDir)
}

func (m *Manager) BuildPrompt(workingDir string, newPrompt string) (string, error) {
	ctx, err := m.LoadContext(workingDir)
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