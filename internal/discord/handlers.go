package discord

import (
	"context"
	"log"
	"strings"
	"tars-bot/internal/discord/voice"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "chat":
			b.handleChatCommand(s, i)
		case "join":
			b.handleJoinCommand(s, i)
		case "leave":
			b.handleLeaveCommand(s, i)
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

func (b *Bot) handleJoinCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if user is in a voice channel
	voiceState, err := s.State.VoiceState(i.GuildID, i.Member.User.ID)
	if err != nil {
		log.Printf("Error getting voice state: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error getting your voice state",
			},
		})
		return
	}

	if voiceState == nil || voiceState.ChannelID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You need to be in a voice channel to use this command",
			},
		})
		return
	}

	// Log channel ID for debugging
	log.Printf("Attempting to join voice channel: %s", voiceState.ChannelID)

	// Check if bot is already in a voice channel
	if _, exists := voice.GetActiveConnection(i.GuildID); exists {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "I'm already in a voice channel",
			},
		})
		return
	}

	// Create new voice connection
	vc, err := voice.NewVoiceConnection(s, i.GuildID, voiceState.ChannelID, b.Agent)
	if err != nil {
		log.Printf("Error creating voice connection: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error creating voice connection",
			},
		})
		return
	}

	// Connect to voice channel
	err = vc.Connect()
	if err != nil {
		log.Printf("Error connecting to voice channel: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error connecting to voice channel: " + err.Error(),
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Joined voice channel!",
		},
	})
}

func (b *Bot) handleLeaveCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if bot is in a voice channel
	conn, exists := voice.GetActiveConnection(i.GuildID)
	if !exists {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "I'm not in a voice channel",
			},
		})
		return
	}

	// Disconnect from voice channel
	err := conn.Disconnect()
	if err != nil {
		log.Printf("Error disconnecting from voice channel: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error disconnecting from voice channel",
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Left voice channel!",
		},
	})
}
