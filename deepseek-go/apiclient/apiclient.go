package apiclient

import (
	"context"
	"fmt"
	"io"      // Added
	"strings" // Added
	"github.com/fatih/color" // Added
	"github.com/sashabaranov/go-openai"
	"github.com/user/deepseek-go/models"
	"github.com/user/deepseek-go/toolbridge"
)

const (
	DefaultBaseURL = "https://api.openai.com/v1"
	DeepSeekBaseURL = "https://api.deepseek.com"
)

type APIClient struct {
	client *openai.Client
}

func NewAPIClient(apiKey string, baseURL string) *APIClient {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL

	client := openai.NewClientWithConfig(config)
	return &APIClient{client: client}
}

// Global color for tool messages, or pass it around.
var toolColor = color.New(color.FgCyan)
var errorColor = color.New(color.FgRed) // For internal errors within apiclient

// --- Local Fallback Definitions for Stream Tool Call Chunking ---
// These are used if the direct library types (openai.ToolCallChunk) are not resolving.

type localFunctionCallStream struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type localToolCallChunk struct {
	Index    *int                     `json:"index"` // Pointer, as in openai.ToolCallChunk
	ID       string                   `json:"id,omitempty"`
	Type     openai.ToolType          `json:"type,omitempty"` // Assuming openai.ToolType resolves
	Function *localFunctionCallStream `json:"function,omitempty"`
}

// Helper to reconstruct ToolCalls from localToolCallChunk.
// Output is []openai.ToolCall (for use in non-stream messages).
func accumulateToolCallChunks(chunks []localToolCallChunk) []openai.ToolCall {
	toolCallMap := make(map[int]*openai.ToolCall) // Keyed by chunk.Index

	for _, chunk := range chunks {
		if chunk.Index == nil {
			errorColor.Println("Warning: localToolCallChunk received with nil Index during streaming.")
			continue
		}
		idx := *chunk.Index

		tc, exists := toolCallMap[idx]
		if !exists {
			// Assuming compiler sees ToolCall.Function as a value type based on persistent error
			tc = &openai.ToolCall{
				ID:   chunk.ID,
				Type: chunk.Type,
				Function: openai.FunctionCall{ // Assign struct value directly
					Name:      "",
					Arguments: "",
				},
			}
			toolCallMap[idx] = tc
		}

		if chunk.ID != "" && tc.ID == "" {
			tc.ID = chunk.ID
		}
		if chunk.Type != "" && tc.Type == "" {
			tc.Type = chunk.Type
		}

		// Assuming compiler sees chunk.Function (*localFunctionCallStream) correctly as a pointer for the local type.
		// The library type for delta.ToolCalls[i].Function is what's problematic if it's seen as value.
		if chunk.Function != nil {
			if chunk.Function.Name != "" {
				tc.Function.Name = chunk.Function.Name // tc.Function is now a value type
			}
			if chunk.Function.Arguments != "" {
				tc.Function.Arguments += chunk.Function.Arguments
			}
		}
	}

	// Convert map to slice
	// The order might not be guaranteed if map iteration order is random, but usually ToolCall IDs are what matter.
	// For processing, ensure tools are processed based on their logical order if necessary.
	// However, the API usually provides chunks for one tool call, then the next.
	result := make([]openai.ToolCall, 0, len(toolCallMap))
	for _, tc := range toolCallMap {
		result = append(result, *tc)
	}
	// It might be better to sort them by index if index is consistently available and matters for order.
	// For now, this conversion is basic.
	return result
}


// SendChatCompletionWithPotentialTools sends a chat completion request, handling potential tool calls and streaming.
func (c *APIClient) SendChatCompletionWithPotentialTools(
	conversationHistory *models.ConversationHistory,
	tools []openai.Tool,
	isRecursiveCall bool,
	assistantColor *color.Color, // Pass the color for printing assistant messages
) (string, error) {
	if conversationHistory == nil || len(conversationHistory.GetMessages()) == 0 {
		return "", fmt.Errorf("conversation history cannot be empty")
	}
	conversationHistory.Trim(models.MaxConversationMessages)

	req := openai.ChatCompletionRequest{
		Model:    "deepseek-chat",
		Messages: conversationHistory.GetMessages(),
		Stream:   true,
	}

	if len(tools) > 0 && !isRecursiveCall {
		req.Tools = tools
	}

	stream, err := c.client.CreateChatCompletionStream(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("CreateChatCompletionStream API error: %w", err)
	}
	defer stream.Close()

	var fullResponseContent strings.Builder
	var accumulatedLocalToolCallChunks []localToolCallChunk // Use local type
	var finalFinishReason openai.FinishReason
	var lastValidResponse openai.ChatCompletionStreamResponse // To store the last valid response for FinishReason

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			if len(lastValidResponse.Choices) > 0 && lastValidResponse.Choices[0].FinishReason != "" {
				finalFinishReason = lastValidResponse.Choices[0].FinishReason
			}
			break
		}
		if err != nil {
			return fullResponseContent.String(), fmt.Errorf("stream Recv error: %w", err)
		}

		if len(response.Choices) > 0 {
			delta := response.Choices[0].Delta
			if delta.Content != "" {
				assistantColor.Print(delta.Content)
				fullResponseContent.WriteString(delta.Content)
			}

			if len(delta.ToolCalls) > 0 {
				for _, libChunk := range delta.ToolCalls { // libChunk is openai.ToolCallChunk
					var currentLocalChunk localToolCallChunk
					currentLocalChunk.Index = libChunk.Index
					currentLocalChunk.ID = libChunk.ID
					currentLocalChunk.Type = libChunk.Type

					// Adapting to compiler error: if libChunk.Function is seen as a value type
					// A non-empty function call usually has a name.
					if libChunk.Function.Name != "" || libChunk.Function.Arguments != "" {
						currentLocalChunk.Function = &localFunctionCallStream{
							Name:      libChunk.Function.Name,
							Arguments: libChunk.Function.Arguments,
						}
					} else {
						currentLocalChunk.Function = nil // Or keep as empty if it was initialized
					}
					accumulatedLocalToolCallChunks = append(accumulatedLocalToolCallChunks, currentLocalChunk)
				}
			}
			lastValidResponse = response
		}
	}

	// After stream is finished
	finalAssistantMessageContent := fullResponseContent.String()
	reconstructedToolCalls := accumulateToolCallChunks(accumulatedLocalToolCallChunks) // Corrected variable name

	// Add the fully formed assistant message (text + tool call requests) to history
	assistantMessageForHistory := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: finalAssistantMessageContent,
	}
	if len(reconstructedToolCalls) > 0 {
		assistantMessageForHistory.ToolCalls = reconstructedToolCalls
	}
	conversationHistory.AddMessage(assistantMessageForHistory)


	if finalFinishReason == openai.FinishReasonToolCalls {
		if len(reconstructedToolCalls) == 0 {
			return finalAssistantMessageContent, fmt.Errorf("finish reason is tool_calls, but no tool calls found after stream processing")
		}

		// Announce tool calls if any were made by the assistant
		if len(reconstructedToolCalls) > 0 {
			if finalAssistantMessageContent != "" { // if there was text before tool calls
				fmt.Println() // Newline after assistant's text part
			}
			toolColor.Printf("🛠️ Assistant wants to use tools:\n")
			for _, tc := range reconstructedToolCalls {
				toolColor.Printf("  - Function: %s, Args: %s\n", tc.Function.Name, tc.Function.Arguments)
			}
		}


		for _, toolCall := range reconstructedToolCalls {
			toolColor.Printf("\n🔄 Executing tool: %s (ID: %s)\n", toolCall.Function.Name, toolCall.ID)
			toolMessage, execErr := toolbridge.ExecuteToolCall(toolCall, conversationHistory) // Pass conversationHistory for potential modification (e.g. edit_file context)
			if execErr != nil {
				errorMsg := fmt.Sprintf("Error executing tool %s (ID: %s): %v", toolCall.Function.Name, toolCall.ID, execErr)
				errorColor.Println(errorMsg) // Print error with color
				toolMessage = openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					ToolCallID: toolCall.ID,
					Content:    errorMsg,
					Name:       toolCall.Function.Name,
				}
			}
			// Print tool result before adding to history for visibility
			toolColor.Printf("  Result (for %s):\n%s\n", toolCall.Function.Name, toolMessage.Content)
			conversationHistory.AddMessage(toolMessage)
		}
		// Recursive call: Pass assistantColor for consistency if further streaming occurs.
		// No tools are passed in the recursive call to prevent loops, API expects results of previous tools.
		return c.SendChatCompletionWithPotentialTools(conversationHistory, nil, true, assistantColor)
	}

	return finalAssistantMessageContent, nil
}
