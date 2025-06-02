package discord

import (
	"github.com/bwmarrin/discordgo"
)

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "chat",
		Description: "Start a conversation with the AI",
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
		Name:        "voice",
		Description: "Start a voice conversation with the AI",
	},
}

func (b *Bot) registerCommands() error {
	// Register guild commands for development
	if b.Config.GuildID != "" {
		_, err := b.Session.ApplicationCommandBulkOverwrite(b.Session.State.User.ID, b.Config.GuildID, commands)
		return err
	}

	// Register global commands for production
	for _, cmd := range commands {
		_, err := b.Session.ApplicationCommandCreate(b.Session.State.User.ID, "", cmd)
		if err != nil {
			return err
		}
	}
	return nil
}
