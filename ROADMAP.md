# Roadmap: Rewriting DeepSeek Engineer in Go

This document outlines the phased approach to rewriting the Python-based `deepseek-eng.py` into a robust, concurrent, and efficient Go application.

## Phase 1: Core Functionality & Foundation

**Objective**: Build the essential features to have a working, interactive AI assistant.

- **[In-Progress] Project Setup**:
  - Initialize Go module (`go mod init`).
  - Create the directory structure (`/cmd`, `/internal`, `/pkg`).
- **[In-Progress] Configuration Management**:
  - Load API keys and settings from a `.env` file.
  - Use a struct for type-safe configuration.
- **[In-Progress] API Client**:
  - Build a client to communicate with the DeepSeek API.
  - Handle `chat/completions` endpoint.
  - Marshal and unmarshal JSON request/response bodies.
- **[In-Progress] Basic CLI**:
  - Create an interactive prompt for user input.
  - Use a simple CLI library (e.g., `go-prompt`).
- **[In-Progress] Function Calling (Core)**:
  - Implement the `read_file`, `create_file`, and `edit_file` functions in Go.
  - Define tool schemas to be sent to the DeepSeek API.
  - Parse tool calls from the API response and execute the corresponding Go functions.
  - Send tool results back to the API.

## Phase 2: Advanced Features & UX

**Objective**: Enhance the user experience and add more complex features.

- **[Not Started] Advanced Function Calling**:
  - Implement `read_multiple_files` and `create_multiple_files`.
  - Implement the `/add` command for pre-loading context.
- **[Not Started] Rich Terminal UI**:
  - Integrate a terminal UI library (e.g., `bubbletea`, `tview`).
  - Display color-coded output for better readability.
  - Show loading spinners and progress indicators.
- **[Not Started] Streaming & Concurrency**:
  - Implement streaming for API responses to show content as it arrives.
  - Use goroutines to handle concurrent operations (e.g., API calls, file I/O).
- **[Not Started] Conversation Management**:
  - Implement conversation history.
  - Add logic to trim the history to prevent token overflow.

## Phase 3: Polish, Testing & Distribution

**Objective**: Make the application robust, reliable, and easy to distribute.

- **[Not Started] Comprehensive Testing**:
  - Add unit tests for all major functions.
  - Implement integration tests for the API client and function calling logic.
- **[Not Started] Error Handling & Reporting**:
  - Implement more granular error handling.
  - Provide user-friendly error messages.
- **[Not Started] UI Refinements**:
  - Implement tables for diff previews.
  - Add more sophisticated UI elements as needed.
- **[Not Started] Build & Distribution**:
  - Create build scripts for different platforms (Linux, macOS, Windows).
  - Write documentation for installation and usage.

## Future Ideas

- **[Not Started] Plugin System**: Allow users to define their own custom tools.
- **[Not Started] Caching**: Cache API responses to reduce latency and cost.
- **[Not Started] Pre-commit Hooks**: Integrate with Git to act as a pre-commit hook for code analysis.
