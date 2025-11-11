package permissions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Permissions struct {
	AllowedTools    []string `json:"allowedTools,omitempty"`
	DisallowedTools []string `json:"disallowedTools,omitempty"`
	PermissionMode  string   `json:"permissionMode,omitempty"`
	AdditionalDirs  []string `json:"additionalDirs,omitempty"`
}

func getPermissionsPath(cwd string) string {
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".claude-acme")
	
	hash := sha256.Sum256([]byte(cwd))
	dirHash := hex.EncodeToString(hash[:])
	permDir := filepath.Join(baseDir, dirHash)
	os.MkdirAll(permDir, 0755)
	return filepath.Join(permDir, "permissions.json")
}

func Read(cwd string) (*Permissions, error) {
	path := getPermissionsPath(cwd)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Permissions{
			AllowedTools:   []string{"Read"},
			PermissionMode: "acceptEdits",
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read permissions file: %w", err)
	}

	var p Permissions
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse settings: %w", err)
	}

	return &p, nil
}

func Write(cwd string, perms *Permissions) error {
	permPath := getPermissionsPath(cwd)
	data, err := json.MarshalIndent(perms, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	err = os.WriteFile(permPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write permissions file: %w", err)
	}

	return nil
}
