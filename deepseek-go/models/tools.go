package models

// PropertyDetail describes a single property in a parameter object.
// It can also be used to describe the items in an array if the Type is "array",
// or the properties of an object if the Type is "object" (for nested structures).
type PropertyDetail struct {
	Type        string                    `json:"type"`
	Description string                    `json:"description,omitempty"`
	Items       *PropertyDetail           `json:"items,omitempty"`      // Used for type "array"
	Properties  map[string]PropertyDetail `json:"properties,omitempty"` // Used for type "object"
	Required    []string                  `json:"required,omitempty"`   // Used for type "object"
}

// ParameterDefinition describes the parameters for a function tool.
type ParameterDefinition struct {
	Type       string                    `json:"type"` // Typically "object"
	Properties map[string]PropertyDetail `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

// FunctionSpecification describes a function tool.
type FunctionSpecification struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Parameters  ParameterDefinition `json:"parameters"`
}

// ToolDefinition is a generic structure for defining a tool for the API.
// Currently, only "function" type is supported by OpenAI.
type ToolDefinition struct {
	Type     string                `json:"type"` // e.g., "function"
	Function FunctionSpecification `json:"function"`
}

// Argument structs for JSON deserialization

// ReadFileArgs arguments for the read_file tool.
type ReadFileArgs struct {
	FilePath string `json:"file_path"`
}

// ReadMultipleFilesArgs arguments for the read_multiple_files tool.
type ReadMultipleFilesArgs struct {
	FilePaths []string `json:"file_paths"`
}

// CreateFileArgs arguments for the create_file tool.
type CreateFileArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// CreateMultipleFilesArgs arguments for the create_multiple_files tool.
// Re-uses FileToCreate from models/files.go
type CreateMultipleFilesArgs struct {
	Files []FileToCreate `json:"files"`
}

// EditFileArgs arguments for the edit_file tool.
type EditFileArgs struct {
	FilePath        string `json:"file_path"`
	OriginalSnippet string `json:"original_snippet"`
	NewSnippet      string `json:"new_snippet"`
}
