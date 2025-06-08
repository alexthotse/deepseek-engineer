package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const MaxFileSize = 5 * 1024 * 1024 // 5MB

// NormalizePath converts a path to an absolute, canonicalized form.
// It also checks for suspicious ".." components after cleaning.
func NormalizePath(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", filePath, err)
	}

	cleanedPath := filepath.Clean(absPath)

	// Check for ".." components that might indicate traversal attempts not fully handled by Clean
	// This is a basic check; more robust validation might be needed depending on security requirements.
	parts := strings.Split(cleanedPath, string(os.PathSeparator))
	for _, part := range parts {
		if part == ".." {
			return "", fmt.Errorf("path traversal attempt detected in %s (cleaned: %s)", filePath, cleanedPath)
		}
	}
	// Ensure the path is within the project directory (assuming /app is project root)
	// This is a placeholder for a more robust check based on actual project root.
	if !strings.HasPrefix(cleanedPath, "/app") {
		return "", fmt.Errorf("path %s (cleaned: %s) is outside the allowed project directory", filePath, cleanedPath)
	}


	return cleanedPath, nil
}

// ReadLocalFile reads the content of a file after normalizing its path.
func ReadLocalFile(filePath string) (string, error) {
	normalizedPath, err := NormalizePath(filePath)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(normalizedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", normalizedPath, err)
	}
	return string(content), nil
}

// CreateFile creates a file with the given content after normalizing its path.
// It creates parent directories if they don't exist.
// It also checks for file size limits.
func CreateFile(filePath string, content string) error {
	if len(content) > MaxFileSize {
		return fmt.Errorf("content for %s exceeds maximum file size of %d bytes", filePath, MaxFileSize)
	}

	normalizedPath, err := NormalizePath(filePath)
	if err != nil {
		return err
	}

	parentDir := filepath.Dir(normalizedPath)
	if err := os.MkdirAll(parentDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create parent directories for %s: %w", normalizedPath, err)
	}

	if err := os.WriteFile(normalizedPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", normalizedPath, err)
	}
	return nil
}

// IsBinaryFile checks if a file is likely a binary file.
// It reads a small chunk and checks for null bytes.
func IsBinaryFile(filePath string) (bool, error) {
	normalizedPath, err := NormalizePath(filePath)
	if err != nil {
		return false, err
	}

	file, err := os.Open(normalizedPath)
	if err != nil {
		return false, fmt.Errorf("failed to open file %s for binary check: %w", normalizedPath, err)
	}
	defer file.Close()

	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err.Error() != "EOF" { // EOF is fine if file is smaller than buffer
		return false, fmt.Errorf("failed to read from file %s for binary check: %w", normalizedPath, err)
	}

	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true, nil // Null byte found, likely binary
		}
	}
	return false, nil // No null bytes found in the first chunk
}

// ApplyDiffEdit applies a snippet replacement to a file.
func ApplyDiffEdit(filePath string, originalSnippet string, newSnippet string) error {
	normalizedPath, err := NormalizePath(filePath) // Ensure path is safe
	if err != nil {
		return err
	}

	// Reading the file directly to avoid re-normalization by ReadLocalFile
	contentBytes, err := os.ReadFile(normalizedPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s for diff edit: %w", normalizedPath, err)
	}
	content := string(contentBytes)

	count := strings.Count(content, originalSnippet)
	if count == 0 {
		return fmt.Errorf("original snippet not found in %s", normalizedPath)
	}
	if count > 1 {
		// For now, log a warning. Future versions might offer strategies for multiple occurrences.
		// log.Printf("Warning: Original snippet found %d times in %s. Replacing only the first.", count, normalizedPath)
		// This logging should be handled by the caller or a proper logging mechanism.
		// For this function, we proceed to replace the first one.
	}

	newContent := strings.Replace(content, originalSnippet, newSnippet, 1)

	// Write the modified content back.
	// CreateFile handles path normalization and directory creation, but we already normalized.
	// Re-using CreateFile might be slightly less efficient due to double normalization,
	// but safer and DRY. Or, call os.WriteFile directly after ensuring parent dir.
	// For simplicity here, let's use os.WriteFile directly as parent dir check is done by CreateFile
	// and we are essentially "editing" an existing file (after read was successful).
	// However, the original file might not have parent dirs if it was created with a different tool.
	// Let's stick to CreateFile for safety and consistency for now.
	// Re-evaluating: CreateFile includes a size check which is good.
	// It also normalizes, which is redundant here.
	// Let's call os.WriteFile directly but ensure parent dir.

	parentDir := filepath.Dir(normalizedPath)
	if err := os.MkdirAll(parentDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create parent directories for %s before writing diff: %w", normalizedPath, err)
	}

	if len(newContent) > MaxFileSize {
		return fmt.Errorf("modified content for %s exceeds maximum file size of %d bytes", normalizedPath, MaxFileSize)
	}

	if err := os.WriteFile(normalizedPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write modified content to %s: %w", normalizedPath, err)
	}

	return nil
}
