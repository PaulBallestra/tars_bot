package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken       string
	OpenAIKey          string
	PostgresConnString string
}

func Load() *Config {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found")
	}

	return &Config{
		DiscordToken:       os.Getenv("DISCORD_TOKEN"),
		OpenAIKey:          os.Getenv("OPENAI_API_KEY"),
		PostgresConnString: os.Getenv("POSTGRES_CONN_STRING"),
	}
}
