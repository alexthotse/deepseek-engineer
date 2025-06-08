package models

// FileToCreate represents a file that needs to be created.
type FileToCreate struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// FileToEdit represents a file that needs to be edited.
// It includes the original snippet to locate the edit position
// and the new snippet to replace it.
type FileToEdit struct {
	Path            string `json:"path"`
	OriginalSnippet string `json:"original_snippet"`
	NewSnippet      string `json:"new_snippet"`
}
