package vectorstore

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

type PostgreSQLVectorStore struct {
	pool *pgxpool.Pool
}

type Conversation struct {
	ID        int64
	GuildID   string
	UserID    string
	SessionID string
	Message   string
	Response  string
	Embedding []float32
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewPostgreSQLVectorStore(connString string) (*PostgreSQLVectorStore, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	err = pool.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize tables if they don't exist
	err = initializeDatabase(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &PostgreSQLVectorStore{pool: pool}, nil
}

func initializeDatabase(pool *pgxpool.Pool) error {
	// Enable pgvector extension if not already enabled
	_, err := pool.Exec(context.Background(), "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("failed to enable vector extension: %w", err)
	}

	// Create conversations table
	_, err = pool.Exec(context.Background(), `
        CREATE TABLE IF NOT EXISTS conversations (
            id BIGSERIAL PRIMARY KEY,
            guild_id VARCHAR(255) NOT NULL,
            user_id VARCHAR(255) NOT NULL,
            session_id VARCHAR(255) NOT NULL,
            message TEXT NOT NULL,
            response TEXT NOT NULL,
            embedding vector(1536),
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        )
    `)
	if err != nil {
		return fmt.Errorf("failed to create conversations table: %w", err)
	}

	// Create index for vector search
	_, err = pool.Exec(context.Background(), `
        CREATE INDEX IF NOT EXISTS conversation_embedding_idx
        ON conversations USING ivfflat (embedding vector_cosine_ops)
    `)
	if err != nil {
		return fmt.Errorf("failed to create vector index: %w", err)
	}

	return nil
}

func (vs *PostgreSQLVectorStore) StoreConversation(ctx context.Context, guildID, userID, sessionID, message, response string, embedding []float32) error {
	query := `
        INSERT INTO conversations
        (guild_id, user_id, session_id, message, response, embedding)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (guild_id, user_id, session_id, message)
        DO UPDATE SET
            response = EXCLUDED.response,
            embedding = EXCLUDED.embedding,
            updated_at = NOW()
    `

	_, err := vs.pool.Exec(ctx, query, guildID, userID, sessionID, message, response, embedding)
	if err != nil {
		return fmt.Errorf("failed to store conversation: %w", err)
	}

	return nil
}

func (vs *PostgreSQLVectorStore) SearchSimilar(ctx context.Context, guildID, userID string, queryEmbedding []float32, limit int) ([]Conversation, error) {
	query := `
        SELECT id, guild_id, user_id, session_id, message, response, embedding, created_at, updated_at
        FROM conversations
        WHERE guild_id = $1 AND user_id = $2
        ORDER BY embedding <=> $3
        LIMIT $4
    `

	rows, err := vs.pool.Query(ctx, query, guildID, userID, pq.Array(queryEmbedding), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search conversations: %w", err)
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var conv Conversation
		err := rows.Scan(
			&conv.ID,
			&conv.GuildID,
			&conv.UserID,
			&conv.SessionID,
			&conv.Message,
			&conv.Response,
			&conv.Embedding,
			&conv.CreatedAt,
			&conv.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

func (vs *PostgreSQLVectorStore) Close() error {
	vs.pool.Close()
	return nil
}
