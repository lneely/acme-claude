package debug

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	a "9fans.net/go/acme"
)

func Tail(ctx context.Context, tw *a.Win) {
	homeDir, _ := os.UserHomeDir()
	debugDir := filepath.Join(homeDir, ".claude", "debug")

	// Get baseline of existing files and their sizes
	baselineFiles := make(map[string]int64)
	files, _ := os.ReadDir(debugDir)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".txt") {
			filePath := filepath.Join(debugDir, file.Name())
			if info, err := os.Stat(filePath); err == nil {
				baselineFiles[filePath] = info.Size()
			}
		}
	}

	tw.Fprintf("body", "[TRACE] Monitoring debug directory with %d existing files\n", len(baselineFiles))

	// Monitor for new content
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	noChangeCount := 0
	totalLines := 0

	for {
		select {
		case <-ctx.Done():
			tw.Fprintf("body", "[TRACE] Debug monitoring stopped, read %d total lines\n", totalLines)
			return
		case <-ticker.C:
			hadChanges := false
			currentFiles, _ := os.ReadDir(debugDir)

			for _, file := range currentFiles {
				if !strings.HasSuffix(file.Name(), ".txt") {
					continue
				}

				filePath := filepath.Join(debugDir, file.Name())
				info, err := os.Stat(filePath)
				if err != nil {
					continue
				}

				currentSize := info.Size()
				baselineSize, exists := baselineFiles[filePath]

				if !exists || currentSize > baselineSize {
					// New file or new content
					lines := readFile(filePath, baselineSize, tw)
					if lines > 0 {
						hadChanges = true
						totalLines += lines
					}
					// Always update baseline when file grows, even if no important lines
					if currentSize > baselineSize {
						hadChanges = true
					}
					baselineFiles[filePath] = currentSize
				}
			}

			if hadChanges {
				noChangeCount = 0
			} else {
				noChangeCount++
				if noChangeCount > 60 {
					tw.Fprintf("body", "[TRACE] No new debug output for 3 seconds, read %d total lines\n", totalLines)
					return
				}
			}
		}
	}
}

func readFile(filePath string, startOffset int64, tw *a.Win) int {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	if startOffset > 0 {
		file.Seek(startOffset, io.SeekStart)
	}

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if isImportant(line) {
			tw.Fprintf("body", "%s\n", line)
			lineCount++
		}
	}

	return lineCount
}

func isImportant(line string) bool {
	// Filter for important tool usage events
	importantPatterns := []string{
		"for tool: Read",
		"for tool: Write",
		"for tool: Edit",
		"for tool: Glob",
		"for tool: Grep",
		"for tool: Bash",
		"for tool: WebSearch",
		"for tool: WebFetch",
		"for tool: Task",
		"for tool: NotebookEdit",
		"for tool: MultiEdit",
		"tool_use",
	}

	for _, pattern := range importantPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}

	return false
}
