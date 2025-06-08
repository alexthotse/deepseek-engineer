package fileutils

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sashabaranov/go-openai"
	// "github.com/user/deepseek-go/fileops" // Was not used, replaced with local funcs or os package
	"github.com/user/deepseek-go/models"
)

var DefaultExcludedFiles = map[string]bool{
	".gitattributes": true,
	".gitignore":     true,
	".env":           true,
	"go.sum":         true,
	"go.mod":         true, // Depending on use case, might want to see this
	"LICENSE":        true,
	"README.md":      true, // Might want to include sometimes
}

var DefaultExcludedExtensions = map[string]bool{
	// Common binary/uninteresting extensions
	".exe":   true, ".dll": true, ".so": true, ".dylib": true,
	".png":   true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true, ".tiff": true,
	".zip":   true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
	".pdf":   true,
	".doc":   true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
	".o":     true, ".obj": true, ".class": true, ".pyc": true,
	".DS_Store":true,
	// Media files
	".mp3": true, ".wav": true, ".aac": true,
	".mp4": true, ".mov": true, ".avi": true, ".mkv": true,
	// VCS specific (though .git is usually a dir)
	".svn": true, ".hg": true,
}

const DefaultMaxFilesToAdd = 50
const DefaultMaxFileSizeToAdd int64 = 1 * 1024 * 1024 // 1MB, smaller than fileops.MaxFileSize for context flooding reasons

// AddDirectoryToConversation walks a directory, reads valid files, and adds their content to conversation history.
// Paths are resolved relative to the current working directory.
func AddDirectoryToConversation(
	directoryPath string,
	conversationHistory *models.ConversationHistory,
	maxFiles int,
	maxFileSize int64,
	excludedFiles map[string]bool,
	excludedExtensions map[string]bool,
) error {
	absDirectoryPath, err := filepath.Abs(directoryPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", directoryPath, err)
	}

	// Check if the directory itself exists
	info, err := os.Stat(absDirectoryPath)
	if err != nil {
		return fmt.Errorf("failed to stat directory %s: %w", absDirectoryPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absDirectoryPath)
	}

	filesAdded := 0
	err = filepath.WalkDir(absDirectoryPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("Warning: Error accessing path %q: %v. Skipping.", path, err)
			return nil // Continue walking even if some parts are inaccessible
		}

		if filesAdded >= maxFiles {
			log.Printf("Reached maximum number of files to add (%d). Stopping.", maxFiles)
			return fs.SkipDir // Stop walking if max files reached (use io.EOF or a custom error if WalkDir supports it for early exit)
			// fs.SkipDir will skip remaining entries in current dir, then continue with next dir. For full stop, need different error.
			// For now, this is a soft limit, it might add a few more from the current directory being processed.
			// A more precise way is to return a specific error like io.EOF and check for it.
		}

		// Skip directories themselves, only process files
		if d.IsDir() {
			// Skip hidden directories (e.g., .git, .vscode)
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." && d.Name() != ".." {
				log.Printf("Skipping hidden directory: %s", path)
				return fs.SkipDir
			}
			// log.Printf("Walking directory: %s", path) // For debugging
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(d.Name(), ".") {
			log.Printf("Skipping hidden file: %s", path)
			return nil
		}

		// Check against excluded files map
		if excludedFiles[d.Name()] {
			log.Printf("Skipping excluded file (by name): %s", path)
			return nil
		}

		// Check against excluded extensions map
		ext := filepath.Ext(d.Name())
		if excludedExtensions[ext] {
			log.Printf("Skipping excluded file (by extension %s): %s", ext, path)
			return nil
		}

		// Check file size
		fileInfo, err := d.Info()
		if err != nil {
			log.Printf("Warning: Failed to get file info for %s: %v. Skipping.", path, err)
			return nil
		}
		if fileInfo.Size() > maxFileSize {
			log.Printf("Skipping large file (%d bytes > %d bytes): %s", fileInfo.Size(), maxFileSize, path)
			return nil
		}
		if fileInfo.Size() == 0 {
			log.Printf("Skipping empty file: %s", path)
			return nil
		}

		// Check if binary file using fileops.IsBinaryFile.
		// Note: IsBinaryFile uses fileops.NormalizePath which is /app prefixed.
		// This will fail if `path` is not under /app.
		// For CLI, we need a version of IsBinaryFile that works with arbitrary CWD-relative paths.
		// Let's assume for now we either:
		// 1. Make fileops.IsBinaryFile more flexible (outside scope of this step)
		// 2. Copy/adapt IsBinaryFile logic here for CLI context.
		// For now, we'll proceed and acknowledge this will be an issue if path isn't under /app.
		// A quick fix could be to temporarily use a simplified binary check here or skip it.
		// Let's use a simplified check for now to avoid NormalizePath issue.
		isBin, err := IsLikelyBinary(path) // Using a local, simplified check - now capitalized
		if err != nil {
			log.Printf("Warning: Failed to check binary status for %s: %v. Skipping.", path, err)
			return nil
		}
		if isBin {
			log.Printf("Skipping binary file: %s", path)
			return nil
		}

		// Read file content using fileops.ReadLocalFile
		// This also uses fileops.NormalizePath and will fail if path isn't under /app.
		// To make this CLI friendly with CWD-relative paths, we should read directly.
		// content, err := fileops.ReadLocalFile(path)
		contentBytes, err := os.ReadFile(path) // Direct read
		if err != nil {
			log.Printf("Warning: Failed to read file %s: %v. Skipping.", path, err)
			return nil
		}
		content := string(contentBytes)


		// Add file content to conversation
		// The path used in the message should be the one relative to the initial directoryPath or an absolute one.
		// Using the absolute path `path` from WalkDir is fine.
		message := fmt.Sprintf("Content of file '%s':\n```\n%s\n```", path, content)
		conversationHistory.AddMessage(openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem, // Or User, depending on how LLM should treat this. System is often for pre-loaded context.
			Content: message,
		})
		log.Printf("Added file to context: %s", path)
		filesAdded++

		return nil
	})

	if err != nil {
		// Check if it's the specific error we used to stop WalkDir
		if err == fs.SkipDir { // This error won't actually be returned by WalkDir itself.
			log.Println("Finished walking directory, max files limit might have been hit in some subdirectories.")
			return nil
		}
		return fmt.Errorf("error walking the path %q: %w", absDirectoryPath, err)
	}
	log.Printf("Finished adding directory content. Total files added: %d", filesAdded)
	return nil
}

// IsLikelyBinary is a simplified local binary check for CLI context
// to avoid issues with fileops.NormalizePath if paths are not under /app.
// Exported for use in main package for the /add command.
func IsLikelyBinary(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to open file %s for binary check: %w", filePath, err)
	}
	defer file.Close()

	buffer := make([]byte, 1024) // Read first 1024 bytes
	n, err := file.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		return false, fmt.Errorf("failed to read from file %s for binary check: %w", filePath, err)
	}

	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true, nil // Null byte found
		}
	}
	return false, nil
}
