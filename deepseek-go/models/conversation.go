package models

import "github.com/sashabaranov/go-openai"

// ConversationHistory is a slice of ChatCompletionMessage, representing the history of a conversation.
type ConversationHistory []openai.ChatCompletionMessage

// AddMessage appends a new message to the conversation history.
func (ch *ConversationHistory) AddMessage(message openai.ChatCompletionMessage) {
	*ch = append(*ch, message)
}

// GetMessages returns the current conversation history.
// This function might seem redundant but can be expanded later if we need
// to add system messages or perform other modifications before returning.
func (ch *ConversationHistory) GetMessages() []openai.ChatCompletionMessage {
	return *ch
}

const MaxConversationMessages = 20 // Default max messages, can be overridden

// Trim reduces the conversation history if it exceeds maxMessages.
// It always keeps the first message (assumed to be the system prompt)
// and the most recent (maxMessages - 1) messages.
func (ch *ConversationHistory) Trim(maxMessages int) {
	if maxMessages <= 1 { // Must keep at least system + 1 user/assistant message
		maxMessages = 2
	}
	history := *ch
	if len(history) <= maxMessages {
		return
	}

	// Keep the first message (system prompt) and the last (maxMessages - 1) messages.
	trimmedHistory := []openai.ChatCompletionMessage{history[0]}         // Keep system prompt
	trimmedHistory = append(trimmedHistory, history[len(history)-(maxMessages-1):]...) // Keep the tail

	*ch = trimmedHistory
}
