package permissions

import (
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

func Read(cwd string) (*Permissions, error) {
	path := filepath.Join(cwd, ".claude-permissions")

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
	permPath := filepath.Join(cwd, ".claude-permissions")
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
