package discord

import (
	"context"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "chat":
			b.handleChatCommand(s, i)
		case "voice":
			b.handleVoiceCommand(s, i)
		}
	}
}

func (b *Bot) mentionHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Check if the bot was mentioned
	if m.Mentions == nil || len(m.Mentions) == 0 {
		return
	}

	for _, mention := range m.Mentions {
		if mention.ID == s.State.User.ID {
			// Remove the mention from the message
			content := m.Content
			for _, mention := range m.Mentions {
				content = strings.ReplaceAll(content, mention.Mention(), "")
			}
			content = strings.TrimSpace(content)

			if content == "" {
				s.ChannelMessageSend(m.ChannelID, "You mentioned me! What would you like to talk about?")
				return
			}

			// Process the message with the AI agent
			response, err := b.Agent.ProcessMessage(context.Background(), m.Author.ID, content)
			if err != nil {
				log.Printf("Error processing message: %v", err)
				s.ChannelMessageSend(m.ChannelID, "Sorry, I had trouble processing that message.")
				return
			}

			s.ChannelMessageSend(m.ChannelID, response)
			return
		}
	}
}

func (b *Bot) handleChatCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	message := options[0].StringValue()

	// Process the message with the AI agent
	response, err := b.Agent.ProcessMessage(context.Background(), i.Member.User.ID, message)
	if err != nil {
		log.Printf("Error processing message: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Sorry, I had trouble processing that message.",
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
		},
	})
}

func (b *Bot) handleVoiceCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Implement voice command logic here
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Voice command received! (Not yet implemented)",
		},
	})
}
