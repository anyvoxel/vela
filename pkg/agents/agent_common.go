// Package agents contains implementation of llm agents.
package agents

import (
	"strings"

	"github.com/cloudwego/eino/schema"
)

func extractMessageText(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	if msg.Content != "" {
		return msg.Content
	}
	if len(msg.AssistantGenMultiContent) == 0 {
		return ""
	}
	parts := make([]string, 0, len(msg.AssistantGenMultiContent))
	for _, part := range msg.AssistantGenMultiContent {
		if part.Type != schema.ChatMessagePartTypeText || part.Text == "" {
			continue
		}
		parts = append(parts, part.Text)
	}
	return strings.Join(parts, "")
}
