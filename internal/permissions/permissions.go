package permissions

import (
	"claude-acme/internal/ui"
	"claude-acme/internal/util"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"9fans.net/go/acme"
)

type Permissions struct {
	AllowedTools    []string `json:"allowedTools,omitempty"`
	DisallowedTools []string `json:"disallowedTools,omitempty"`
	PermissionMode  string   `json:"permissionMode,omitempty"`
	AdditionalDirs  []string `json:"additionalDirs,omitempty"`
}

var AllTools = []string{
	"Read", "Write", "Edit", "MultiEdit", "NotebookEdit",
	"Glob", "Grep", "Bash", "BashOutput", "KillBash",
	"WebSearch", "WebFetch", "Task", "TodoWrite", "ExitPlanMode",
	"Bash(git:*)", "Bash(mkdir:*)", "Bash(ls:*)", "Bash(cd:*)",
	"Bash(cp:*)", "Bash(mv:*)", "Bash(rm:*)", "Bash(chmod:*)",
}

func (p *Permissions) GetAllowed() []string {
	return p.AllowedTools
}

func (p *Permissions) GetDisallowed() []string {
	allowedSet := make(map[string]bool)
	for _, tool := range p.AllowedTools {
		allowedSet[tool] = true
	}

	var disallowed []string
	for _, tool := range AllTools {
		if !allowedSet[tool] {
			disallowed = append(disallowed, tool)
		}
	}

	for _, tool := range p.DisallowedTools {
		if !slices.Contains(disallowed, tool) {
			disallowed = append(disallowed, tool)
		}
	}

	return disallowed
}

func (p *Permissions) Allow(tool string) {
	if !slices.Contains(p.AllowedTools, tool) {
		p.AllowedTools = append(p.AllowedTools, tool)
	}
	p.DisallowedTools = remove(p.DisallowedTools, tool)
}

func (p *Permissions) Deny(tool string) {
	if !slices.Contains(p.DisallowedTools, tool) {
		p.DisallowedTools = append(p.DisallowedTools, tool)
	}
	p.AllowedTools = remove(p.AllowedTools, tool)
}

func (p *Permissions) Remove(tool string) {
	p.AllowedTools = remove(p.AllowedTools, tool)
	p.DisallowedTools = remove(p.DisallowedTools, tool)
}

func remove(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

func GetPermissionsPath(cwd string) string {
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".claude-acme")

	hash := sha256.Sum256([]byte(cwd))
	dirHash := hex.EncodeToString(hash[:])
	permDir := filepath.Join(baseDir, dirHash)
	os.MkdirAll(permDir, 0755)
	return filepath.Join(permDir, "permissions.json")
}

func Read(cwd string) (*Permissions, error) {
	path := GetPermissionsPath(cwd)

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
	data, err := json.MarshalIndent(perms, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	path := GetPermissionsPath(cwd)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write permissions file: %w", err)
	}

	return nil
}

func Run() {
	w, err := ui.WindowOpen(filepath.Join(util.Getwd(), "+Claude-Permissions"))
	if err != nil {
		fmt.Printf("Couldn't create permissions window: %v\n", err)
		return
	}
	ui.TagSet(w, "Show Edit Save")
	ui.WindowDirty(w, false)

	showCurrent(w)

	for e := range w.EventChan() {
		switch e.C2 {
		case 'x', 'X':
			switch string(e.Text) {
			case "Del":
				w.Ctl("delete")
				return
			case "Show":
				showCurrent(w)
			case "Edit":
				showEdit(w)
			case "Save":
				save(w)
			case "default", "plan", "acceptEdits", "bypassPermissions":
				setMode(w, string(e.Text))
			default:
				w.WriteEvent(e)
			}
		case 'l', 'L':
			w.Ctl("clean")
		}
	}
}

func showCurrent(w *acme.Win) {
	cwd := util.Getwd()
	perms, err := Read(cwd)
	if err != nil {
		w.Fprintf("body", "Error loading permissions: %v\n", err)
		return
	}

	w.Clear()
	w.Fprintf("body", "# Active permissions for: %s\n", cwd)

	mode := perms.PermissionMode
	if mode == "" {
		mode = "default"
	}
	w.Fprintf("body", "# PermissionMode: %s\n\n", mode)

	modes := []string{"default", "plan", "acceptEdits", "bypassPermissions"}
	w.Fprintf("body", "Mode: ")
	for _, m := range modes {
		w.Fprintf("body", "[%s] ", m)
	}
	w.Fprintf("body", "\n\n")

	for _, tool := range perms.AllowedTools {
		w.Fprintf("body", "+ %s\n", tool)
	}
	for _, tool := range perms.DisallowedTools {
		w.Fprintf("body", "- %s\n", tool)
	}

	w.Ctl("clean")
}

func showEdit(w *acme.Win) {
	cwd := util.Getwd()
	perms, err := Read(cwd)
	if err != nil {
		w.Fprintf("body", "Error loading permissions: %v\n", err)
		return
	}

	allowed := make(map[string]bool)
	for _, tool := range perms.AllowedTools {
		allowed[tool] = true
	}

	w.Clear()
	w.Fprintf("body", "# Available tools to grant - edit with + to allow, - to deny, ~ to remove\n\n")

	for _, tool := range AllTools {
		if !allowed[tool] {
			w.Fprintf("body", "  %s\n", tool)
		}
	}

	w.Ctl("clean")
}

func save(w *acme.Win) {
	content, err := w.ReadAll("body")
	if err != nil {
		w.Fprintf("body", "Error reading window: %v\n", err)
		return
	}

	cwd := util.Getwd()
	perms, err := Read(cwd)
	if err != nil {
		w.Fprintf("body", "Error loading permissions: %v\n", err)
		return
	}

	allow, deny, remove := parseEdits(string(content))
	for _, tool := range allow {
		perms.Allow(tool)
	}
	for _, tool := range deny {
		perms.Deny(tool)
	}
	for _, tool := range remove {
		perms.Remove(tool)
	}

	if err := Write(cwd, perms); err != nil {
		w.Fprintf("body", "Error saving permissions: %v\n", err)
		return
	}

	w.Fprintf("body", "\n✓ Permissions updated successfully!\n")
	showCurrent(w)
}

func parseEdits(content string) (allow, deny, remove []string) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "+") {
			if tool := strings.TrimSpace(line[1:]); tool != "" {
				allow = append(allow, tool)
			}
		} else if strings.HasPrefix(line, "-") {
			if tool := strings.TrimSpace(line[1:]); tool != "" {
				deny = append(deny, tool)
			}
		} else if strings.HasPrefix(line, "~") {
			if tool := strings.TrimSpace(line[1:]); tool != "" {
				remove = append(remove, tool)
			}
		}
	}
	return
}

func setMode(w *acme.Win, mode string) {
	cwd := util.Getwd()
	perms, err := Read(cwd)
	if err != nil {
		w.Fprintf("body", "Error loading permissions: %v\n", err)
		return
	}

	if mode == "default" {
		perms.PermissionMode = ""
	} else {
		perms.PermissionMode = mode
	}

	if err := Write(cwd, perms); err != nil {
		w.Fprintf("body", "Error saving permissions: %v\n", err)
		return
	}

	showCurrent(w)
}
