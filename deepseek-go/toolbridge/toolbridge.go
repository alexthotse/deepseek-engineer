package toolbridge

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/user/deepseek-go/fileops"
	"github.com/user/deepseek-go/models"
)

// GetToolDefinitions returns the list of tool definitions for the OpenAI API.
func GetToolDefinitions() []openai.Tool {
	tools := []models.ToolDefinition{
		{
			Type: "function",
			Function: models.FunctionSpecification{
				Name:        "read_file",
				Description: "Reads the content of a single file. The file path should be an absolute path within the project.",
				Parameters: models.ParameterDefinition{
					Type: "object",
					Properties: map[string]models.PropertyDetail{
						"file_path": {Type: "string", Description: "The absolute path to the file to be read."},
					},
					Required: []string{"file_path"},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionSpecification{
				Name:        "read_multiple_files",
				Description: "Reads the content of multiple files. File paths should be absolute paths within the project.",
				Parameters: models.ParameterDefinition{
					Type: "object",
					Properties: map[string]models.PropertyDetail{
						"file_paths": {
							Type:        "array",
							Description: "A list of absolute file paths to be read.",
							Items:       &models.PropertyDetail{Type: "string"},
						},
					},
					Required: []string{"file_paths"},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionSpecification{
				Name:        "create_file",
				Description: "Creates a new file with the given content. The file path should be an absolute path. Overwrites if the file already exists.",
				Parameters: models.ParameterDefinition{
					Type: "object",
					Properties: map[string]models.PropertyDetail{
						"file_path": {Type: "string", Description: "The absolute path where the file will be created."},
						"content":   {Type: "string", Description: "The content to write into the file."},
					},
					Required: []string{"file_path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionSpecification{
				Name: "create_multiple_files",
				Description: "Creates multiple new files with the given content. Each file path should be an absolute path. Overwrites if files already exist.",
				Parameters: models.ParameterDefinition{
					Type: "object",
					Properties: map[string]models.PropertyDetail{
						"files": {
							Type:        "array",
							Description: "A list of files to create.",
							Items: &models.PropertyDetail{ // Defines the structure of each item in the "files" array
								Type: "object",
								Properties: map[string]models.PropertyDetail{
									"path":    {Type: "string", Description: "Absolute path for the file."},
									"content": {Type: "string", Description: "Content for the file."},
								},
								Required: []string{"path", "content"},
							},
						},
					},
					Required: []string{"files"},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionSpecification{
				Name:        "edit_file",
				Description: "Edits an existing file by replacing an original snippet with a new snippet. The file path should be an absolute path.",
				Parameters: models.ParameterDefinition{
					Type: "object",
					Properties: map[string]models.PropertyDetail{
						"file_path":         {Type: "string", Description: "The absolute path to the file to be edited."},
						"original_snippet":  {Type: "string", Description: "The exact snippet of text to be replaced."},
						"new_snippet":       {Type: "string", Description: "The new snippet of text to replace the original."},
					},
					Required: []string{"file_path", "original_snippet", "new_snippet"},
				},
			},
		},
	}

	openaiTools := make([]openai.Tool, len(tools))
	for i, t := range tools {
		// Convert our ParameterDefinition.Properties (map[string]PropertyDetail)
		// to json.RawMessage for openai.FunctionDefinition.Parameters
		paramsBytes, err := json.Marshal(t.Function.Parameters)
		if err != nil {
			// This should ideally not happen if our structs are correct.
			// Handle error appropriately in a real application.
			// For now, let's panic or log fatally as it's a setup issue.
			panic(fmt.Sprintf("Error marshalling parameters for tool %s: %v", t.Function.Name, err))
		}

		openaiTools[i] = openai.Tool{
			Type: openai.ToolType(t.Type),
			Function: &openai.FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  json.RawMessage(paramsBytes),
			},
		}
	}
	return openaiTools
}

// ExecuteToolCall processes a single tool call from the API.
// It deserializes arguments, calls the appropriate fileops function,
// and formats the result or error as an openai.ChatCompletionMessage.
// It also updates the conversation history if a file is read by edit_file for the first time.
func ExecuteToolCall(toolCall openai.ToolCall, conversationHistory *models.ConversationHistory) (openai.ChatCompletionMessage, error) {
	var resultText string
	var execError error

	switch toolCall.Function.Name {
	case "read_file":
		var args models.ReadFileArgs
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		if err != nil {
			execError = fmt.Errorf("error unmarshalling arguments for %s: %w", toolCall.Function.Name, err)
			break
		}
		content, err := fileops.ReadLocalFile(args.FilePath)
		if err != nil {
			execError = fmt.Errorf("error executing %s: %w", toolCall.Function.Name, err)
		} else {
			resultText = fmt.Sprintf("Content of file %s:\n```\n%s\n```", args.FilePath, content)
		}
	case "read_multiple_files":
		var args models.ReadMultipleFilesArgs
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		if err != nil {
			execError = fmt.Errorf("error unmarshalling arguments for %s: %w", toolCall.Function.Name, err)
			break
		}
		var results []string
		var encounteredError bool
		for _, fp := range args.FilePaths {
			content, err := fileops.ReadLocalFile(fp)
			if err != nil {
				results = append(results, fmt.Sprintf("Error reading %s: %v", fp, err))
				encounteredError = true // Mark that at least one error occurred
			} else {
				results = append(results, fmt.Sprintf("Content of file %s:\n```\n%s\n```", fp, content))
			}
		}
		resultText = strings.Join(results, "\n\n")
		if encounteredError { // If any file read failed, the overall operation is marked as partially failed.
			// execError = fmt.Errorf("one or more files could not be read during %s", toolCall.Function.Name)
			// No, we return the mixed results. The LLM can decide.
		}

	case "create_file":
		var args models.CreateFileArgs
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		if err != nil {
			execError = fmt.Errorf("error unmarshalling arguments for %s: %w", toolCall.Function.Name, err)
			break
		}
		err = fileops.CreateFile(args.FilePath, args.Content)
		if err != nil {
			execError = fmt.Errorf("error executing %s: %w", toolCall.Function.Name, err)
		} else {
			resultText = fmt.Sprintf("File %s created successfully.", args.FilePath)
		}
	case "create_multiple_files":
		var args models.CreateMultipleFilesArgs
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		if err != nil {
			execError = fmt.Errorf("error unmarshalling arguments for %s: %w", toolCall.Function.Name, err)
			break
		}
		var results []string
		var encounteredError bool
		for _, f := range args.Files {
			err := fileops.CreateFile(f.Path, f.Content)
			if err != nil {
				results = append(results, fmt.Sprintf("Error creating %s: %v", f.Path, err))
				encounteredError = true
			} else {
				results = append(results, fmt.Sprintf("File %s created successfully.", f.Path))
			}
		}
		resultText = strings.Join(results, "\n")
		if encounteredError {
			// execError = fmt.Errorf("one or more files could not be created during %s", toolCall.Function.Name)
		}

	case "edit_file":
		var args models.EditFileArgs
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		if err != nil {
			execError = fmt.Errorf("error unmarshalling arguments for %s: %w", toolCall.Function.Name, err)
			break
		}

		// "ensure_file_in_context" logic:
		// Check if file content is already in history. This is a simplified check.
		// A more robust check would parse previous tool calls and their results.
		alreadyInContext := false
		for _, msg := range conversationHistory.GetMessages() {
			if msg.Role == openai.ChatMessageRoleTool && strings.Contains(msg.Content, fmt.Sprintf("Content of file %s:\n", args.FilePath)) {
				alreadyInContext = true
				break
			}
		}
		if !alreadyInContext {
			// log.Printf("File %s not in context for edit, reading it first.", args.FilePath) // Use proper logging
			content, readErr := fileops.ReadLocalFile(args.FilePath)
			if readErr != nil {
				// If we can't read it, we can't add it to context. The edit will likely fail too.
				// execError = fmt.Errorf("error reading file %s for edit context: %w", args.FilePath, readErr)
				// break
				// Let the edit proceed and fail, it will give a clearer error to the LLM.
			} else {
				contextMsg := openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleTool,
					Content: fmt.Sprintf("File %s was not in context. Current content:\n```\n%s\n```", args.FilePath, content),
					// ToolCallID is not strictly for this "pre-emptive" read, but helpful for tracing.
					// However, standard practice is to only add messages that are direct results of a tool_call_id.
					// For now, let's add it to the general conversation history without a tool_call_id.
					// This part of the logic might need refinement based on how the LLM reacts.
					// A cleaner way might be for the LLM to explicitly call read_file first if unsure.
					// For now, we add it directly to the history.
				}
				// This modification of conversationHistory here might be tricky if it's a copy.
				// Ensure conversationHistory is a pointer.
				conversationHistory.AddMessage(contextMsg)
				// We can also prepend this to the resultText of the edit_file call later.
			}
		}

		err = fileops.ApplyDiffEdit(args.FilePath, args.OriginalSnippet, args.NewSnippet)
		if err != nil {
			execError = fmt.Errorf("error executing %s: %w", toolCall.Function.Name, err)
		} else {
			resultText = fmt.Sprintf("File %s edited successfully.", args.FilePath)
		}

	default:
		execError = fmt.Errorf("unknown tool call: %s", toolCall.Function.Name)
	}

	finalContent := resultText
	if execError != nil {
		finalContent = fmt.Sprintf("Error in tool %s: %v", toolCall.Function.Name, execError)
	}

	return openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		ToolCallID: toolCall.ID,
		Content:    finalContent,
		Name:       toolCall.Function.Name,
	}, nil // The error here is for the ExecuteToolCall function itself, not the tool execution.
}
