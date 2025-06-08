package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color" // Added for colorized output
	"github.com/sashabaranov/go-openai"
	"github.com/user/deepseek-go/apiclient"
	"github.com/user/deepseek-go/config"
	"github.com/user/deepseek-go/fileutils"
	"github.com/user/deepseek-go/models"
	"github.com/user/deepseek-go/prompts"
	"github.com/user/deepseek-go/toolbridge"
)

// Define color objects
var (
	userColor      = color.New(color.FgBlue, color.Bold)
	assistantColor = color.New(color.FgGreen)
	systemColor    = color.New(color.FgYellow)
	errorColor     = color.New(color.FgRed)
	// toolColor is defined in apiclient for now, or could be passed around / made global
)

func main() {
	// Disable log prefixes (date/time) for cleaner CLI output
	log.SetFlags(0)

	config.LoadConfig()
	if config.APIKey == "" {
		errorColor.Println("DEEPSEEK_API_KEY not found. Please set it in your environment. Example: export DEEPSEEK_API_KEY=\"your_key_here\"")
		errorColor.Println("Exiting: API key is required for this test.")
		return
	}
	systemColor.Println("DEEPSEEK_API_KEY loaded.")

	apiClient := apiclient.NewAPIClient(config.APIKey, apiclient.DeepSeekBaseURL)
	tools := toolbridge.GetToolDefinitions()

	// Initialize conversation history
	history := models.ConversationHistory{}
	history.AddMessage(openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: prompts.SystemPrompt,
	})
	// No initial user message, user will provide via CLI.

	systemColor.Println("DeepSeek Go CLI initialized. Type 'exit' or 'quit' to end.")

	reader := bufio.NewReader(os.Stdin)
	for {
		userColor.Print("🔵 You> ") // Use color for prompt
		userInput, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				systemColor.Println("Exiting...")
				break
			}
			errorColor.Printf("Error reading input: %v\n", err)
			continue
		}

		userInput = strings.TrimSpace(userInput)

		if userInput == "exit" || userInput == "quit" {
			systemColor.Println("Exiting...")
			break
		}

		if strings.TrimSpace(userInput) == "" {
			continue
		}

		if strings.HasPrefix(userInput, "/add ") {
			pathToAdd := strings.TrimSpace(strings.TrimPrefix(userInput, "/add "))
			if pathToAdd == "" {
				errorColor.Println("Usage: /add <file_path_or_directory_path>")
				continue
			}

			absPath, errAbs := filepath.Abs(pathToAdd)
			if errAbs != nil {
				errorColor.Printf("Error getting absolute path for %s: %v\n", pathToAdd, errAbs)
				continue
			}
			cleanedPath := filepath.Clean(absPath)

			fileInfo, errStat := os.Stat(cleanedPath)
			if errStat != nil {
				errorColor.Printf("Error accessing path %s: %v\n", cleanedPath, errStat)
				continue
			}

			if fileInfo.IsDir() {
				systemColor.Printf("Adding directory %s to conversation context...\n", cleanedPath)
				errAddDir := fileutils.AddDirectoryToConversation(
					cleanedPath,
					&history,
					fileutils.DefaultMaxFilesToAdd,
					fileutils.DefaultMaxFileSizeToAdd,
					fileutils.DefaultExcludedFiles,
					fileutils.DefaultExcludedExtensions,
				)
				if errAddDir != nil {
					errorColor.Printf("Error adding directory %s: %v\n", cleanedPath, errAddDir)
				} else {
					systemColor.Printf("Directory %s processed and added to context.\n", cleanedPath)
				}
			} else { // It's a file
				systemColor.Printf("Adding file %s to conversation context...\n", cleanedPath)
				if fileInfo.Size() > fileutils.DefaultMaxFileSizeToAdd {
					errorColor.Printf("Skipping large file (%d bytes > %d bytes): %s\n", fileInfo.Size(), fileutils.DefaultMaxFileSizeToAdd, cleanedPath)
					continue
				}
				if fileInfo.Size() == 0 {
					systemColor.Printf("Skipping empty file: %s\n", cleanedPath)
					continue
				}
				isBin, binCheckErr := fileutils.IsLikelyBinary(cleanedPath)
				if binCheckErr != nil {
					errorColor.Printf("Error checking if file %s is binary: %v\n", cleanedPath, binCheckErr)
					continue
				}
				if isBin {
					systemColor.Printf("Skipping binary file: %s\n", cleanedPath)
					continue
				}

				contentBytes, errRead := os.ReadFile(cleanedPath)
				if errRead != nil {
					errorColor.Printf("Error reading file %s: %v\n", cleanedPath, errRead)
					continue
				}
				content := string(contentBytes)
				message := fmt.Sprintf("Content of file '%s':\n```\n%s\n```", cleanedPath, content)
				history.AddMessage(openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem, // Or User
					Content: message,
				})
				systemColor.Printf("File %s added to context.\n", cleanedPath)
			}
			continue // Skip sending this /add command itself to the LLM, just go to next prompt
		}

		// Add user message to history
		history.AddMessage(openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: userInput,
		})

		// Call API
		assistantColor.Print("🤖 Assistant> ") // Print prefix for streamed response
		// The 'false' indicates it's not a recursive call initially.
		// SendChatCompletionWithPotentialTools will now handle printing the streamed response.
		_, err = apiClient.SendChatCompletionWithPotentialTools(&history, tools, false, assistantColor) // Pass color
		if err != nil {
			errorColor.Printf("\nAPI Error: %v\n", err)
		} else {
			fmt.Println() // Add a newline after the streamed response is complete
		}

		// Optional: Print full history for debugging, can be removed later
		// log.Println("\n--- Current Conversation History ---")
		// for _, msg := range history.GetMessages() {
		// 	log.Printf("[%s]: %s", msg.Role, msg.Content)
		// 	if len(msg.ToolCalls) > 0 {
		// 		for _, tc := range msg.ToolCalls {
		// 			log.Printf("  Tool Call ID: %s, Function: %s, Args: %s", tc.ID, tc.Function.Name, tc.Function.Arguments)
		// 		}
		// 	}
		// }
		// log.Println("--- End History ---")

	}
}
