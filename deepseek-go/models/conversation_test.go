package models

import (
	"testing"
	"github.com/sashabaranov/go-openai"
	"reflect"
)

func TestConversationHistory_Trim(t *testing.T) {
	sysPrompt := openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: "System"}
	userMsg := func(i int) openai.ChatCompletionMessage {
		return openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: "User " + string(rune(i))}
	}
	assistantMsg := func(i int) openai.ChatCompletionMessage {
		return openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, Content: "Assistant " + string(rune(i))}
	}

	tests := []struct {
		name          string
		initialHistory ConversationHistory
		maxMessages   int
		expectedHistory ConversationHistory
	}{
		{
			name: "No trimming needed, history shorter than max",
			initialHistory: ConversationHistory{sysPrompt, userMsg(1)},
			maxMessages:   5,
			expectedHistory: ConversationHistory{sysPrompt, userMsg(1)},
		},
		{
			name: "No trimming needed, history equal to max",
			initialHistory: ConversationHistory{sysPrompt, userMsg(1), assistantMsg(1), userMsg(2), assistantMsg(2)},
			maxMessages:   5,
			expectedHistory: ConversationHistory{sysPrompt, userMsg(1), assistantMsg(1), userMsg(2), assistantMsg(2)},
		},
		{
			name: "Trim needed, simple case",
			// Sys, U1, A1, U2, A2, U3 (len 6) -> trim to 4 (Sys, A2, U3)
			initialHistory: ConversationHistory{sysPrompt, userMsg(1), assistantMsg(1), userMsg(2), assistantMsg(2), userMsg(3)},
			maxMessages:   4,
			// Expected: Sys, U2, A2, U3 (Error in manual trace, should be Sys + last 3: Sys, A2, U3)
			// Corrected Expected: history[0] and history[len-max+1:] = history[0], history[6-4+1 = 3:] = history[0], history[3], history[4], history[5]
			// Sys, userMsg(2), assistantMsg(2), userMsg(3)
			expectedHistory: ConversationHistory{sysPrompt, userMsg(2), assistantMsg(2), userMsg(3)},
		},
		{
			name: "Trim needed, keep only system and one latest message",
			initialHistory: ConversationHistory{sysPrompt, userMsg(1), assistantMsg(1), userMsg(2)},
			maxMessages:   2, // System + 1 latest
			expectedHistory: ConversationHistory{sysPrompt, userMsg(2)},
		},
		{
			name: "Trim with maxMessages 1 (should default to 2)",
			initialHistory: ConversationHistory{sysPrompt, userMsg(1), assistantMsg(1), userMsg(2)},
			maxMessages:   1, // Will be treated as 2
			expectedHistory: ConversationHistory{sysPrompt, userMsg(2)},
		},
		{
			name: "Trim with maxMessages 0 (should default to 2)",
			initialHistory: ConversationHistory{sysPrompt, userMsg(1), assistantMsg(1), userMsg(2)},
			maxMessages:   0, // Will be treated as 2
			expectedHistory: ConversationHistory{sysPrompt, userMsg(2)},
		},
		{
			name: "History with only system prompt, no trim",
			initialHistory: ConversationHistory{sysPrompt},
			maxMessages: 5,
			expectedHistory: ConversationHistory{sysPrompt},
		},
		{
			name: "History with only system prompt, trim to 2 (no change)",
			initialHistory: ConversationHistory{sysPrompt},
			maxMessages: 2,
			expectedHistory: ConversationHistory{sysPrompt},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := tt.initialHistory // Create a copy to avoid modifying the slice in the test struct
			history.Trim(tt.maxMessages)
			if !reflect.DeepEqual(history, tt.expectedHistory) {
				t.Errorf("Trim() got = %v, want %v", history, tt.expectedHistory)
				// For debugging:
				t.Logf("Got length: %d, Expected length: %d", len(history), len(tt.expectedHistory))
				for i := 0; i < len(history) || i < len(tt.expectedHistory); i++ {
					var gotMsg, wantMsg openai.ChatCompletionMessage
					if i < len(history) { gotMsg = history[i] }
					if i < len(tt.expectedHistory) { wantMsg = tt.expectedHistory[i] }
					if !reflect.DeepEqual(gotMsg, wantMsg) {
						t.Logf("Mismatch at index %d: Got %v, Want %v", i, gotMsg, wantMsg)
					}
				}
			}
		})
	}
}
