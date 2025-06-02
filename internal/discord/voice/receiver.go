// internal/discord/voice/receiver.go
package voice

import (
	"bytes"
	"log"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type AudioReceiver struct {
	Connection *VoiceConnection
	Buffer     bytes.Buffer
	Mutex      sync.Mutex
}

func NewAudioReceiver(vc *VoiceConnection) *AudioReceiver {
	return &AudioReceiver{
		Connection: vc,
	}
}

func (ar *AudioReceiver) Start() {
	log.Println("Starting audio receiver")

	opusBuffer := make([]byte, 0, 20*960) // Buffer for 20ms chunks at 48kHz
	opusChan := make(chan *discordgo.Packet, 10)

	// Enable receiving Opus packets
	ar.Connection.VoiceConnection.OpusRecv = opusChan
	ar.Connection.VoiceConnection.Speaking(true)
	defer ar.Connection.VoiceConnection.Speaking(false)

	for {
		select {
		case <-ar.Connection.Context.Done():
			return
		case packet := <-opusChan:
			if packet == nil {
				continue
			}
			ar.Mutex.Lock()
			opusBuffer = append(opusBuffer, packet.Opus...)
			ar.Mutex.Unlock()

			// Process when we have enough data (20ms chunks)
			if len(opusBuffer) >= 960 {
				ar.processAudioChunk(opusBuffer)
				opusBuffer = opusBuffer[:0] // Reset buffer
			}
		}
	}
}

func (ar *AudioReceiver) processAudioChunk(opusData []byte) {
	// Send to STT
	text, err := ar.Connection.Agent.STT.Transcribe(ar.Connection.Context, opusData)
	if err != nil {
		log.Printf("Error transcribing audio: %v", err)
		return
	}

	if text == "" {
		log.Println("Empty transcription received")
		return
	}

	log.Printf("Transcribed text: %s", text)

	// Process with AI agent
	response, err := ar.Connection.Agent.ProcessMessage(ar.Connection.Context, "voice-user", text)
	if err != nil {
		log.Printf("Error processing message: %v", err)
		return
	}

	// Send response to TTS
	ar.Connection.AudioSender.QueueResponse(response)
}
