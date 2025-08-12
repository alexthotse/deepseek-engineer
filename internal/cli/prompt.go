package cli

import (
	"fmt"
	"os"

	"deepseek-eng-go/internal/agent"
	"deepseek-eng-go/internal/api"
	"github.com/c-bata/go-prompt"
)

const systemPrompt = `You are an elite software engineer called DeepSeek Engineer with decades of experience across all programming domains.
Your expertise spans system design, algorithms, testing, and best practices.
You provide thoughtful, well-structured solutions while explaining your reasoning.

Core capabilities:
1. Code Analysis & Discussion
   - Analyze code with expert-level insight
   - Explain complex concepts clearly
   - Suggest optimizations and best practices
   - Debug issues with precision

2. File Operations (via function calls):
   - read_file: Read a single file's content
   - create_file: Create or overwrite a single file
   - edit_file: Make precise edits to existing files using snippet replacement

Guidelines:
1. Provide natural, conversational responses explaining your reasoning
2. Use function calls when you need to read or modify files
3. For file operations:
   - Always read files first before editing them to understand the context
   - Use precise snippet matching for edits
   - Explain what changes you're making and why
   - Consider the impact of changes on the overall codebase
4. Follow language-specific best practices
5. Suggest tests or validation steps when appropriate
6. Be thorough in your analysis and recommendations`

// runner holds the state for the CLI runner
type runner struct {
	apiClient *api.Client
	messages  []api.Message
}

// Executor is the main loop that processes user input.
func (r *runner) Executor(in string) {
	if in == "exit" || in == "quit" {
		fmt.Println("Goodbye!")
		os.Exit(0)
		return
	}

	// Add user message to history
	r.messages = append(r.messages, api.Message{Role: "user", Content: in})

	// Loop until we get a response without tool calls
	for {
		fmt.Println("🤖 Sending to DeepSeek...")
		resp, err := r.apiClient.ChatCompletion(r.messages)
		if err != nil {
			fmt.Printf("API call failed: %v\n", err)
			// Remove the last user message to allow retrying
			r.messages = r.messages[:len(r.messages)-1]
			return
		}

		if len(resp.Choices) == 0 {
			fmt.Println("Assistant> I have no response for that.")
			return
		}

		assistantMessage := resp.Choices[0].Message
		r.messages = append(r.messages, assistantMessage)

		if len(assistantMessage.ToolCalls) > 0 {
			fmt.Printf("Assistant wants to use %d tool(s)...\n", len(assistantMessage.ToolCalls))

			toolResults := []api.Message{}
			for _, toolCall := range assistantMessage.ToolCalls {
				fmt.Printf("  -> Executing tool: %s\n", toolCall.Function.Name)

				result, err := agent.ExecuteToolCall(toolCall)
				if err != nil {
					fmt.Printf("  -> Error executing tool %s: %v\n", toolCall.Function.Name, err)
					toolResults = append(toolResults, api.Message{
						Role:       "tool",
						ToolCallID: toolCall.ID,
						Content:    fmt.Sprintf("Error: %v", err),
					})
				} else {
					fmt.Printf("  -> Tool %s executed successfully.\n", toolCall.Function.Name)
					toolResults = append(toolResults, api.Message{
						Role:       "tool",
						ToolCallID: toolCall.ID,
						Content:    result,
					})
				}
			}
			// Add all tool results to the conversation history
			r.messages = append(r.messages, toolResults...)
			// Continue the loop to send tool results back to the API
			continue
		}

		// No tool calls, this is the final response.
		fmt.Println("Assistant> " + assistantMessage.Content)
		break // Exit the loop
	}
}

// Completer provides suggestions for auto-completion.
func Completer(d prompt.Document) []prompt.Suggest {
	// We can add suggestions for "exit", "quit", etc. later
	return []prompt.Suggest{}
}

// Run starts the interactive prompt.
func Run(apiClient *api.Client) {
	r := &runner{
		apiClient: apiClient,
		messages: []api.Message{
			{Role: "system", Content: systemPrompt},
		},
	}

	p := prompt.New(
		r.Executor,
		Completer,
		prompt.OptionTitle("deepseek-eng-go"),
		prompt.OptionPrefix("You> "),
		prompt.OptionPrefixTextColor(prompt.Blue),
	)
	p.Run()
}
