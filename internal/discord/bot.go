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

	// Configure intents - specifically for voice support
	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent // Enable to access message content

	// Enable voice-specific features
	session.StateEnabled = true
	session.State.TrackVoice = true // Ensure voice state tracking is enabled

	log.Printf("Discord session created with intents: %d", session.Identify.Intents)
	log.Printf("Voice state tracking enabled: %v", session.State.TrackVoice)

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
	b.Session.AddHandler(b.voiceStateUpdateHandler) // Track voice state changes

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

// Track voice state changes to help with debugging
func (b *Bot) voiceStateUpdateHandler(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	// Only log our own voice state updates or if someone joins/leaves our channel
	if v.UserID == s.State.User.ID {
		log.Printf("Bot voice state updated - ChannelID: %s, Self Mute: %v, Self Deaf: %v",
			v.ChannelID, v.SelfMute, v.SelfDeaf)
	} else {
		// Log when users join or leave voice channels that we're in
		for _, guild := range s.State.Guilds {
			for _, vs := range guild.VoiceStates {
				if vs.UserID == s.State.User.ID && vs.ChannelID == v.ChannelID {
					// We're in this channel, so this update is relevant
					action := "updated their voice state in"
					if v.ChannelID == "" {
						action = "left"
					}
					user, _ := s.User(v.UserID)
					username := v.UserID
					if user != nil {
						username = user.Username
					}
					log.Printf("User %s %s our voice channel - Self Mute: %v, Self Deaf: %v",
						username, action, v.SelfMute, v.SelfDeaf)
					break
				}
			}
		}
	}
}

func (b *Bot) Close() {
	b.Session.Close()
}
