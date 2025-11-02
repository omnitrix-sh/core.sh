package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ListDirTool struct {
	workDir string
}

func NewListDirTool(workDir string) *ListDirTool {
	return &ListDirTool{
		workDir: workDir,
	}
}

func (t *ListDirTool) Name() string {
	return "list_dir"
}

func (t *ListDirTool) Description() string {
	return `List contents of a directory to explore the project structure.

Usage:
- Provide directory path (defaults to current directory)
- Optionally show hidden files
- Optionally show full details

Use this to understand project organization before reading or modifying files.`
}

func (t *ListDirTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"dir_path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to list (defaults to current directory)",
			},
			"show_hidden": map[string]interface{}{
				"type":        "boolean",
				"description": "Include hidden files (starting with .)",
			},
		},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	dirPath := GetStringArg(args, "dir_path", ".")
	showHidden := GetBoolArg(args, "show_hidden", false)

	// Resolve path
	if !filepath.IsAbs(dirPath) {
		dirPath = filepath.Join(t.workDir, dirPath)
	}

	// Security check
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	absWorkDir, err := filepath.Abs(t.workDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve work directory: %w", err)
	}

	if !strings.HasPrefix(absPath, absWorkDir) {
		return "", fmt.Errorf("access denied: path is outside working directory")
	}

	// Check if directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory not found: %s", dirPath)
		}
		return "", fmt.Errorf("failed to stat directory: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", dirPath)
	}

	// Read directory
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Directory: %s\n", dirPath))
	output.WriteString(fmt.Sprintf("Entries: %d\n\n", len(entries)))

	var dirs, files []string

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files if requested
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}

		if entry.IsDir() {
			dirs = append(dirs, name+"/")
		} else {
			// Get file size
			info, err := entry.Info()
			size := ""
			if err == nil {
				size = formatSize(info.Size())
			}
			files = append(files, fmt.Sprintf("%-40s %s", name, size))
		}
	}

	// Print directories first
	if len(dirs) > 0 {
		output.WriteString("Directories:\n")
		for _, dir := range dirs {
			output.WriteString(fmt.Sprintf("  %s\n", dir))
		}
		output.WriteString("\n")
	}

	// Then files
	if len(files) > 0 {
		output.WriteString("Files:\n")
		for _, file := range files {
			output.WriteString(fmt.Sprintf("  %s\n", file))
		}
	}

	return output.String(), nil
}

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.1f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}
