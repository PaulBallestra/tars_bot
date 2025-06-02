package voice

import (
	"bytes"
	"io"
	"log"
)

type AudioReceiver struct {
	Connection *VoiceConnection
	Buffer     bytes.Buffer
}

func NewAudioReceiver(vc *VoiceConnection) *AudioReceiver {
	return &AudioReceiver{
		Connection: vc,
	}
}

func (ar *AudioReceiver) Start() {
	log.Println("Starting audio receiver")

	// Configure voice connection
	ar.Connection.VoiceConnection.Speaking(true)
	defer ar.Connection.VoiceConnection.Speaking(false)

	// Create a buffer for audio data
	buffer := make([][]byte, 0)

	// Receive audio packets
	for {
		select {
		case <-ar.Connection.Context.Done():
			return
		default:
			packet, err := ar.Connection.VoiceConnection.Receive()
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Printf("Error receiving audio packet: %v", err)
				continue
			}

			// Process the audio packet
			buffer = append(buffer, packet.Opus)
			ar.processAudio(buffer)
			buffer = buffer[:0] // Reset buffer
		}
	}
}

func (ar *AudioReceiver) processAudio(buffer [][]byte) {
	// Combine audio packets
	opusData := bytes.Join(buffer, []byte{})

	// Convert Opus to PCM (simplified - in production you'd use a proper decoder)
	pcmData, err := opusToPCM(opusData)
	if err != nil {
		log.Printf("Error converting Opus to PCM: %v", err)
		return
	}

	// Send to STT
	text, err := ar.Connection.Agent.STT.Transcribe(ar.Connection.Context, pcmData)
	if err != nil {
		log.Printf("Error transcribing audio: %v", err)
		return
	}

	if text == "" {
		return
	}

	log.Printf("Transcribed text: %s", text)

	// Process with AI agent
	response, err := ar.Connection.Agent.ProcessMessage(ar.Connection.Context, "voice-user", text)
	if err != nil {
		log.Printf("Error processing message: %v", err)
		return
	}

	// Store conversation in vector database
	err = ar.Connection.Agent.Memory.StoreConversation("voice-user", text, response)
	if err != nil {
		log.Printf("Error storing conversation: %v", err)
	}

	// Send response to TTS
	ar.Connection.AudioSender.QueueResponse(response)
}

// Simplified Opus to PCM conversion (in production use a proper library)
func opusToPCM(opusData []byte) ([]byte, error) {
	// This is a placeholder - in production you would:
	// 1. Use a proper Opus decoder
	// 2. Convert to PCM format expected by your STT service
	// 3. Handle sample rates and channels properly

	// For testing, we'll just return the raw data
	return opusData, nil
}
