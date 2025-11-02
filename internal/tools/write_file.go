package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type WriteFileTool struct {
	workDir string
}

func NewWriteFileTool(workDir string) *WriteFileTool {
	return &WriteFileTool{
		workDir: workDir,
	}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return `Write content to a file, creating it if it doesn't exist or overwriting if it does.

Usage:
- Provide the file path (relative to working directory or absolute)
- Provide the content to write
- Optionally create parent directories

Use this to create new files or modify existing ones. Always read the file first before modifying to avoid conflicts.`
}

func (t *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to write (relative or absolute)",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
			"create_dirs": map[string]interface{}{
				"type":        "boolean",
				"description": "Create parent directories if they don't exist (default: true)",
			},
		},
		"required": []string{"file_path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	filePath := GetStringArg(args, "file_path", "")
	if filePath == "" {
		return "", fmt.Errorf("file_path is required")
	}

	content := GetStringArg(args, "content", "")
	createDirs := GetBoolArg(args, "create_dirs", true)

	// Resolve path
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(t.workDir, filePath)
	}

	// Security check: ensure path is within work directory
	absPath, err := filepath.Abs(filePath)
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

	// Check if it's a directory
	if info, err := os.Stat(absPath); err == nil && info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	// Create parent directories if needed
	if createDirs {
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create parent directories: %w", err)
		}
	}

	// Read existing content for comparison
	var oldContent string
	if existingContent, err := os.ReadFile(absPath); err == nil {
		oldContent = string(existingContent)
		if oldContent == content {
			return fmt.Sprintf("File %s already contains the exact content. No changes made.", filePath), nil
		}
	}

	// Write file
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Generate response
	var response strings.Builder
	if oldContent == "" {
		response.WriteString(fmt.Sprintf("Created file: %s\n", filePath))
		response.WriteString(fmt.Sprintf("Lines written: %d\n", strings.Count(content, "\n")+1))
	} else {
		response.WriteString(fmt.Sprintf("Modified file: %s\n", filePath))
		oldLines := strings.Count(oldContent, "\n") + 1
		newLines := strings.Count(content, "\n") + 1
		response.WriteString(fmt.Sprintf("Lines: %d -> %d (%+d)\n", oldLines, newLines, newLines-oldLines))
	}

	return response.String(), nil
}
