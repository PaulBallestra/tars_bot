package discord

import (
	"log"

	"tars-bot/internal/ai"
	"tars-bot/internal/config"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session *discordgo.Session
	Agent   *ai.AIAgent
	Config  *config.Config
}

func NewBot(cfg *config.Config, agent *ai.AIAgent) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, err
	}

	// Configure intents
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions

	return &Bot{
		Session: session,
		Agent:   agent,
		Config:  cfg,
	}, nil
}

func (b *Bot) Start() error {
	// Register handlers
	b.Session.AddHandler(b.readyHandler)
	b.Session.AddHandler(b.mentionHandler)
	b.Session.AddHandler(b.interactionHandler)

	// Open the websocket connection
	err := b.Session.Open()
	if err != nil {
		return err
	}

	// Register commands
	err = b.registerCommands()
	if err != nil {
		log.Printf("Error registering commands: %v", err)
	}

	return nil
}

func (b *Bot) readyHandler(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
}

func (b *Bot) Close() {
	b.Session.Close()
}
