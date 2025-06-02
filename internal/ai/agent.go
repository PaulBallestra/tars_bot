package ai

import (
	"context"
	"fmt"
	"log"
	"strings"

	"tars-bot/internal/ai/openai"
)

type AIAgent struct {
	Memory *Memory
	STT    *openai.STT
	TTS    *openai.TTS
}

func NewAIAgent(apiKey string) *AIAgent {
	return &AIAgent{
		Memory: NewMemory(),
		STT:    openai.NewSTT(apiKey),
		TTS:    openai.NewTTS(apiKey),
	}
}

func (a *AIAgent) ProcessMessage(ctx context.Context, userID, message string) (string, error) {
	// Retrieve relevant context from memory
	context := a.Memory.Retrieve(userID, message)

	// Combine with new message
	prompt := fmt.Sprintf("Context:\n%s\n\nUser Message: %s", context, message)
	log.Printf("Processing message for user %s: %s", userID, message)

	// Call OpenAI API
	response, err := openai.CallChatCompletion(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI: %w", err)
	}

	// Clean up the response
	response = strings.TrimSpace(response)
	if response == "" {
		return "I'm not sure how to respond to that. Could you elaborate?", nil
	}

	// Update memory
	a.Memory.Store(userID, message, response)

	return response, nil
}
