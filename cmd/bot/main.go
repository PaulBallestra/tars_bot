package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"tars-bot/internal/ai"
	"tars-bot/internal/config"
	"tars-bot/internal/discord"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize AI agent with PostgreSQL connection
	agent, err := ai.NewAIAgent(cfg.OpenAIKey, os.Getenv("POSTGRES_CONN_STRING"))
	if err != nil {
		log.Fatalf("Failed to initialize AI agent: %v", err)
	}
	defer agent.Close()

	// Initialize Discord bot
	bot, err := discord.NewBot(cfg, agent)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Start the bot
	err = bot.Start()
	if err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}
	defer bot.Session.Close()

	log.Println("Bot is now running. Press CTRL+C to exit.")

	// Wait for termination signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
