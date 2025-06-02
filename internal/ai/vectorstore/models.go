package vectorstore

import (
	"context"
	"time"
)

type Conversation struct {
	ID        int
	GuildID   string
	UserID    string
	SessionID string
	Message   string
	Response  string
	Embedding []float32
	CreatedAt time.Time
	UpdatedAt time.Time
}

type VectorStore interface {
	StoreConversation(ctx context.Context, guildID, userID, sessionID, message, response string, embedding []float32) error
	SearchSimilar(ctx context.Context, guildID, userID string, queryEmbedding []float32, limit int) ([]Conversation, error)
	Close() error
}
