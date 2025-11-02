package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxFileSize = 10 * 1024 * 1024 // 10MB

type ReadFileTool struct {
	workDir string
}

func NewReadFileTool(workDir string) *ReadFileTool {
	return &ReadFileTool{
		workDir: workDir,
	}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return `Read the contents of a file from the filesystem. Use this to examine source code, configuration files, or any text-based files.

Usage:
- Provide the file path (relative to working directory or absolute)
- Optionally specify line range to read partial content

The tool will return the file contents with line numbers for easy reference.`
}

func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to read (relative or absolute)",
			},
			"start_line": map[string]interface{}{
				"type":        "integer",
				"description": "Optional: Line number to start reading from (1-indexed)",
			},
			"end_line": map[string]interface{}{
				"type":        "integer",
				"description": "Optional: Line number to stop reading at (inclusive)",
			},
		},
		"required": []string{"file_path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	filePath := GetStringArg(args, "file_path", "")
	if filePath == "" {
		return "", fmt.Errorf("file_path is required")
	}

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

	// Check if file exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", filePath)
		}
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	// Check file size
	if info.Size() > maxFileSize {
		return "", fmt.Errorf("file too large (%d bytes, max %d bytes)", info.Size(), maxFileSize)
	}

	// Read file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	
	// Handle line range
	startLine := GetIntArg(args, "start_line", 1)
	endLine := GetIntArg(args, "end_line", len(lines))

	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine {
		return "", fmt.Errorf("start_line (%d) must be <= end_line (%d)", startLine, endLine)
	}

	// Format output with line numbers
	var output strings.Builder
	output.WriteString(fmt.Sprintf("File: %s\n", filePath))
	output.WriteString(fmt.Sprintf("Lines: %d-%d of %d\n\n", startLine, endLine, len(lines)))

	for i := startLine - 1; i < endLine; i++ {
		output.WriteString(fmt.Sprintf("%4d | %s\n", i+1, lines[i]))
	}

	return output.String(), nil
}
