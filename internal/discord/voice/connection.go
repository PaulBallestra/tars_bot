package voice

import (
	"context"
	"errors"
	"log"
	"sync"

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

	// Join voice channel with proper settings
	voiceConn, err := vc.Session.ChannelVoiceJoin(vc.GuildID, vc.ChannelID, false, true)
	if err != nil {
		return err
	}
	vc.VoiceConnection = voiceConn

	// Initialize audio receiver
	vc.AudioReceiver = NewAudioReceiver(vc)
	go vc.AudioReceiver.Start()

	// Initialize audio sender
	vc.AudioSender, err = NewAudioSender(vc)
	if err != nil {
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
