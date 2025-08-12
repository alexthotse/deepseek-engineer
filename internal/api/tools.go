package api

// availableTools defines the list of tools available to the DeepSeek API.
var availableTools = []Tool{
	{
		Type: "function",
		Function: Function{
			Name:        "read_file",
			Description: "Read the content of a single file from the filesystem",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]string{
						"type":        "string",
						"description": "The path to the file to read",
					},
				},
				"required": []string{"file_path"},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "create_file",
			Description: "Create a new file or overwrite an existing file with the provided content",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]string{
						"type":        "string",
						"description": "The path where the file should be created",
					},
					"content": map[string]string{
						"type":        "string",
						"description": "The content to write to the file",
					},
				},
				"required": []string{"file_path", "content"},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "edit_file",
			Description: "Edit an existing file by replacing a specific snippet with new content",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]string{
						"type":        "string",
						"description": "The path to the file to edit",
					},
					"original_snippet": map[string]string{
						"type":        "string",
						"description": "The exact text snippet to find and replace",
					},
					"new_snippet": map[string]string{
						"type":        "string",
						"description": "The new text to replace the original snippet with",
					},
				},
				"required": []string{"file_path", "original_snippet", "new_snippet"},
			},
		},
	},
}
