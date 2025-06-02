package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) registerCommands() error {
	// Register global commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "chat",
			Description: "Chat with the AI",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "Your message to the AI",
					Required:    true,
				},
			},
		},
		{
			Name:        "join",
			Description: "Join a voice channel",
			Type:        discordgo.ChatApplicationCommand,
		},
		{
			Name:        "leave",
			Description: "Leave the voice channel",
			Type:        discordgo.ChatApplicationCommand,
		},
	}

	// Register commands globally
	registeredCommands, err := b.Session.ApplicationCommandBulkOverwrite(b.Session.State.User.ID, "", commands)
	if err != nil {
		return err
	}

	log.Printf("Registered %d commands", len(registeredCommands))
	return nil
}
