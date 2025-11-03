package voice

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"tars-bot/internal/ai"

	"github.com/bwmarrin/discordgo"
)

var (
	activeConnections = make(map[string]*VoiceConnection)
	connectionsMutex  sync.Mutex
)

type VoiceConnection struct {
	Session         *discordgo.Session
	GuildID         string
	ChannelID       string
	VoiceConnection *discordgo.VoiceConnection
	AudioReceiver   *AudioReceiver
	AudioSender     *AudioSender
	Agent           *ai.AIAgent
	Context         context.Context
	Cancel          context.CancelFunc
	Mutex           sync.Mutex
}

func NewVoiceConnection(s *discordgo.Session, guildID, channelID string, agent *ai.AIAgent) (*VoiceConnection, error) {
	connectionsMutex.Lock()
	defer connectionsMutex.Unlock()

	if conn, exists := activeConnections[guildID]; exists {
		return conn, errors.New("voice connection already exists for this guild")
	}

	ctx, cancel := context.WithCancel(context.Background())

	vc := &VoiceConnection{
		Session:   s,
		GuildID:   guildID,
		ChannelID: channelID,
		Agent:     agent,
		Context:   ctx,
		Cancel:    cancel,
	}

	activeConnections[guildID] = vc
	return vc, nil
}

func (vc *VoiceConnection) Connect() error {
	vc.Mutex.Lock()
	defer vc.Mutex.Unlock()

	if vc.VoiceConnection != nil {
		return errors.New("already connected")
	}

	log.Printf("Attempting to join voice channel %s in guild %s", vc.ChannelID, vc.GuildID)

	// Make sure we have proper intents set up
	if vc.Session.Identify.Intents&discordgo.IntentsGuildVoiceStates == 0 {
		log.Println("WARNING: Voice state intents are not enabled, adding them now")
		vc.Session.Identify.Intents |= discordgo.IntentsGuildVoiceStates
	}

	// Join with both speaking and listening enabled (mute=false, deaf=false)
	voiceConn, err := vc.Session.ChannelVoiceJoin(vc.GuildID, vc.ChannelID, false, false)
	if err != nil {
		log.Printf("Failed to join voice channel: %v", err)
		return err
	}
	vc.VoiceConnection = voiceConn

	log.Println("Successfully joined voice channel")
	log.Printf("Voice connection ready state: %v", voiceConn.Ready)

	// Wait for connection to stabilize
	waitTime := 3 * time.Second
	log.Printf("Waiting %v for voice connection to stabilize...", waitTime)
	time.Sleep(waitTime)

	// Check if we're actually in the voice channel
	inVoiceChannel := false
	guild, err := vc.Session.State.Guild(vc.GuildID)
	if err == nil {
		for _, vs := range guild.VoiceStates {
			if vs.UserID == vc.Session.State.User.ID && vs.ChannelID == vc.ChannelID {
				inVoiceChannel = true
				log.Printf("Confirmed bot is in voice channel: %s", vc.ChannelID)
				break
			}
		}
	}

	if !inVoiceChannel {
		log.Printf("WARNING: Bot does not appear to be in the voice channel according to guild state")
	}

	// Initialize audio receiver
	log.Println("Initializing audio receiver")
	vc.AudioReceiver = NewAudioReceiver(vc)
	go vc.AudioReceiver.Start()

	// Initialize audio sender
	log.Println("Initializing audio sender")
	vc.AudioSender, err = NewAudioSender(vc)
	if err != nil {
		log.Printf("Error initializing audio sender: %v", err)
		vc.VoiceConnection.Disconnect()
		return err
	}
	go vc.AudioSender.Start()

	return nil
}

func (vc *VoiceConnection) Disconnect() error {
	vc.Mutex.Lock()
	defer vc.Mutex.Unlock()

	vc.Cancel()

	if vc.VoiceConnection != nil {
		err := vc.VoiceConnection.Disconnect()
		if err != nil {
			log.Printf("Error disconnecting voice connection: %v", err)
		}
		vc.VoiceConnection = nil
	}

	connectionsMutex.Lock()
	delete(activeConnections, vc.GuildID)
	connectionsMutex.Unlock()

	return nil
}

func GetActiveConnection(guildID string) (*VoiceConnection, bool) {
	connectionsMutex.Lock()
	defer connectionsMutex.Unlock()

	conn, exists := activeConnections[guildID]
	return conn, exists
}
