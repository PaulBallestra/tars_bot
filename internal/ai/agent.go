package ai

import (
	"context"
	"fmt"
	"log"
	"sync"

	"tars-bot/internal/ai/openai"
	"tars-bot/internal/ai/vectorstore"
)

type AIAgent struct {
	Memory         *Memory
	STT            *openai.STT
	TTS            *openai.TTS
	VectorStore    *vectorstore.PostgreSQLVectorStore
	EmbeddingModel *openai.EmbeddingModel
	Mutex          sync.Mutex
}

func NewAIAgent(apiKey, dbConnString string) (*AIAgent, error) {
	// Initialize vector store
	vectorStore, err := vectorstore.NewPostgreSQLVectorStore(dbConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize vector store: %w", err)
	}

	// Initialize embedding model
	embeddingModel := openai.NewEmbeddingModel(apiKey)

	return &AIAgent{
		Memory:         NewMemory(),
		STT:            openai.NewSTT(apiKey),
		TTS:            openai.NewTTS(apiKey),
		VectorStore:    vectorStore,
		EmbeddingModel: embeddingModel,
	}, nil
}

func (a *AIAgent) ProcessMessage(ctx context.Context, userID, message string) (string, error) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	// Get relevant context from vector store
	embedding, err := a.EmbeddingModel.CreateEmbedding(ctx, message)
	if err != nil {
		log.Printf("Error creating embedding: %v", err)
	}

	var contextMessages []string
	if embedding != nil {
		conversations, err := a.VectorStore.SearchSimilar(ctx, "guild-id", userID, embedding, 3)
		if err != nil {
			log.Printf("Error searching similar conversations: %v", err)
		}

		for _, conv := range conversations {
			contextMessages = append(contextMessages,
				fmt.Sprintf("User: %s\nBot: %s", conv.Message, conv.Response))
		}
	}

	// Combine with new message
	prompt := fmt.Sprintf("Context:\n%s\n\nUser Message: %s", contextMessages, message)

	// Call OpenAI API
	response, err := openai.CallChatCompletion(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI: %w", err)
	}

	// Store conversation in vector database
	if embedding != nil {
		err = a.VectorStore.StoreConversation(
			ctx,
			"guild-id",
			userID,
			"session-id",
			message,
			response,
			embedding,
		)
		if err != nil {
			log.Printf("Error storing conversation: %v", err)
		}
	}

	return response, nil
}

func (a *AIAgent) Close() error {
	if a.VectorStore != nil {
		return a.VectorStore.Close()
	}
	return nil
}
