package ide

import "testing"

func TestToolsChat_PublicTypes(t *testing.T) {
	input := ChatSendInput{SessionID: "session-1", Message: "hello"}
	message := ChatMessage{Role: "user", Content: input.Message}
	out := ChatHistoryOutput{Messages: []ChatMessage{message}}
	if input.SessionID != "session-1" || out.Messages[0].Content != "hello" {
		t.Fatalf("unexpected chat type values: %+v %+v", input, out)
	}
}
