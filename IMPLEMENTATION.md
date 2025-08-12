# Go Implementation Details - Phase 1

This document provides technical details about the Go implementation of the DeepSeek Engineer.

## Project Structure

The project will follow a standard Go project layout:

```
/
├── cmd/
│   └── deepseek-eng/
│       └── main.go         # Main application entry point
├── internal/
│   ├── api/                # Logic for interacting with the DeepSeek API
│   │   └── client.go
│   ├── cli/                # Interactive command-line interface
│   │   └── prompt.go
│   └── platform/           # Cross-platform functionalities
│       └── file_ops.go     # File system operations (read, create, edit)
├── pkg/
│   └── config/             # Configuration loading and management
│       └── config.go
├── .env                    # Environment variables (ignored by git)
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
└── ROADMAP.md              # Project roadmap
└── IMPLEMENTATION.md       # This file
```

## Phase 1 Implementation Details

### 1. Configuration (`/pkg/config/`)

- **File**: `config.go`
- **Library**: `github.com/joho/godotenv` for loading `.env` files.
- **Details**:
  - A `Config` struct will hold the application configuration, starting with the `DeepSeekAPIKey`.
  - A `LoadConfig()` function will read from the environment or a `.env` file and return a `Config` struct.

### 2. DeepSeek API Client (`/internal/api/`)

- **File**: `client.go`
- **Details**:
  - `Client` struct will manage communication with the DeepSeek API.
  - It will use Go's standard `net/http` package for making requests.
  - Structs will be defined to match the JSON structure of the DeepSeek API for `chat/completions`. This includes messages, tools, and tool calls.

### 3. Command-Line Interface (`/internal/cli/`)

- **File**: `prompt.go`
- **Library**: `github.com/c-bata/go-prompt` as a starting point for an interactive CLI.
- **Details**:
  - A `Run()` function will start the interactive prompt.
  - It will handle user input and pass it to the main application logic.

### 4. File Operations (`/internal/platform/`)

- **File**: `file_ops.go`
- **Details**:
  - Go functions for `ReadFile`, `CreateFile`, and `EditFile`.
  - These will be implemented using Go's standard library (`os`, `io/ioutil`).
  - They will include error handling and path normalization.

### 5. Main Application (`/cmd/deepseek-eng/`)

- **File**: `main.go`
- **Details**:
  - The main entry point of the application.
  - It will initialize the configuration and the API client.
  - It will start the CLI and manage the main application loop.
  - It will orchestrate the calls between the CLI, the API client, and the file operations.
