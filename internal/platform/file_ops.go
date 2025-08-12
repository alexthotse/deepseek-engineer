package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadFile reads the content of a file.
func ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("could not read file %s: %w", path, err)
	}
	return string(content), nil
}

// CreateFile creates a new file with content, creating parent directories if needed.
func CreateFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("could not write file %s: %w", path, err)
	}
	return nil
}

// EditFile finds a specific snippet in a file and replaces it.
// For safety, it will only perform the edit if the snippet appears exactly once.
func EditFile(path, originalSnippet, newSnippet string) error {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file for editing %s: %w", path, err)
	}
	content := string(contentBytes)

	occurrences := strings.Count(content, originalSnippet)
	if occurrences == 0 {
		return fmt.Errorf("original snippet not found in file %s", path)
	}
	if occurrences > 1 {
		return fmt.Errorf("ambiguous edit: snippet found %d times in %s", occurrences, path)
	}

	updatedContent := strings.Replace(content, originalSnippet, newSnippet, 1)

	if err := os.WriteFile(path, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("could not write updated content to file %s: %w", path, err)
	}
	return nil
}
