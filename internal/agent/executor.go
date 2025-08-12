package agent

import (
	"encoding/json"
	"fmt"

	"deepseek-eng-go/internal/api"
	"deepseek-eng-go/internal/platform"
)

// ExecuteToolCall takes a tool call from the API and executes the corresponding function.
func ExecuteToolCall(toolCall api.ToolCall) (string, error) {
	switch toolCall.Function.Name {
	case "read_file":
		var args struct {
			FilePath string `json:"file_path"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("error decoding read_file args: %w", err)
		}
		// platform.ReadFile returns the content and an error.
		return platform.ReadFile(args.FilePath)

	case "create_file":
		var args struct {
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("error decoding create_file args: %w", err)
		}
		err := platform.CreateFile(args.FilePath, args.Content)
		if err != nil {
			return "", err // Return the error from the platform function.
		}
		return fmt.Sprintf("Successfully created file: %s", args.FilePath), nil

	case "edit_file":
		var args struct {
			FilePath        string `json:"file_path"`
			OriginalSnippet string `json:"original_snippet"`
			NewSnippet      string `json:"new_snippet"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("error decoding edit_file args: %w", err)
		}
		err := platform.EditFile(args.FilePath, args.OriginalSnippet, args.NewSnippet)
		if err != nil {
			return "", err // Return the error from the platform function.
		}
		return fmt.Sprintf("Successfully edited file: %s", args.FilePath), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
	}
}
