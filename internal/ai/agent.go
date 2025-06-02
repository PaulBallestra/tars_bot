package ai

import (
	"context"
	"log"
	"tars-bot/internal/ai/openai"
	"tars-bot/internal/ai/vectorstore"
)

type AIAgent struct {
	Chat   *openai.ChatClient
	STT    *openai.STTClient
	TTS    *openai.TTSClient
	Memory *vectorstore.PostgreSQLVectorStore
}

func NewAIAgent(openAIKey, postgresConnString string) (*AIAgent, error) {
	// Initialize OpenAI clients
	chatClient := openai.NewChatClient(openAIKey)
	sttClient := openai.NewSTTClient(openAIKey)
	ttsClient := openai.NewTTSClient(openAIKey)

	// Initialize vector store
	vectorStore, err := vectorstore.NewPostgreSQLVectorStore(postgresConnString)
	if err != nil {
		return nil, err
	}

	return &AIAgent{
		Chat:   chatClient,
		STT:    sttClient,
		TTS:    ttsClient,
		Memory: vectorStore,
	}, nil
}

func (a *AIAgent) ProcessMessage(ctx context.Context, userID, message string) (string, error) {
	// Get relevant context from memory
	embedding, err := a.Chat.CreateEmbedding(ctx, message)
	if err != nil {
		return "", err
	}

	// Search for similar conversations
	conversations, err := a.Memory.SearchSimilar(ctx, "", userID, embedding, 3)
	if err != nil {
		log.Printf("Error searching similar conversations: %v", err)
	}

	// Build context from past conversations
	contextMessages := ""
	for _, conv := range conversations {
		contextMessages += "User: " + conv.Message + "\n" +
			"Bot: " + conv.Response + "\n\n"
	}

	// Generate response with context
	prompt := "Context from previous conversations:\n" + contextMessages +
		"\nCurrent conversation:\nUser: " + message + "\nBot:"

	response, err := a.Chat.Completion(ctx, prompt)
	if err != nil {
		return "", err
	}

	// Store the conversation
	embedding, err = a.Chat.CreateEmbedding(ctx, message)
	if err != nil {
		log.Printf("Error creating embedding: %v", err)
	} else {
		err = a.Memory.StoreConversation(ctx, "", userID, "", message, response, embedding)
		if err != nil {
			log.Printf("Error storing conversation: %v", err)
		}
	}

	return response, nil
}

func (a *AIAgent) Close() error {
	if a.Memory != nil {
		return a.Memory.Close()
	}
	return nil
}
