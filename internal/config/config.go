package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken string
	OpenAIKey    string
	GuildID      string // For command registration
}

func Load() *Config {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found")
	}

	return &Config{
		DiscordToken: os.Getenv("DISCORD_TOKEN"),
		OpenAIKey:    os.Getenv("OPENAI_API_KEY"),
		GuildID:      os.Getenv("DISCORD_GUILD_ID"), // Optional for development
	}
}
