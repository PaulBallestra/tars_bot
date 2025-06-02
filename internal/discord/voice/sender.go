package voice

import (
	"log"
	"sync"
)

type AudioSender struct {
	Connection *VoiceConnection
	Queue      chan string
	Mutex      sync.Mutex
}

func NewAudioSender(vc *VoiceConnection) *AudioSender {
	return &AudioSender{
		Connection: vc,
		Queue:      make(chan string, 10),
	}
}

func (as *AudioSender) Start() {
	log.Println("Starting audio sender")

	for {
		select {
		case <-as.Connection.Context.Done():
			return
		case text := <-as.Queue:
			as.processText(text)
		}
	}
}

func (as *AudioSender) QueueResponse(text string) {
	select {
	case as.Queue <- text:
	default:
		log.Println("Audio sender queue full, dropping message")
	}
}

func (as *AudioSender) processText(text string) {
	as.Mutex.Lock()
	defer as.Mutex.Unlock()

	// Generate audio from text
	audioData, err := as.Connection.Agent.TTS.Generate(text)
	if err != nil {
		log.Printf("Error generating TTS: %v", err)
		return
	}

	// Convert to Opus (simplified)
	opusData, err := pcmToOpus(audioData)
	if err != nil {
		log.Printf("Error converting to Opus: %v", err)
		return
	}

	// Send audio in chunks
	chunkSize := 20 * 960 // 20ms chunks at 48kHz
	for i := 0; i < len(opusData); i += chunkSize {
		end := i + chunkSize
		if end > len(opusData) {
			end = len(opusData)
		}

		chunk := opusData[i:end]
		as.Connection.VoiceConnection.Speaking(true)
		err := as.Connection.VoiceConnection.SendOpus(chunk)
		if err != nil {
			log.Printf("Error sending audio: %v", err)
			return
		}
	}

	as.Connection.VoiceConnection.Speaking(false)
}

// Simplified PCM to Opus conversion (in production use a proper library)
func pcmToOpus(pcmData []byte) ([]byte, error) {
	// This is a placeholder - in production you would:
	// 1. Use a proper Opus encoder
	// 2. Convert from PCM to Opus format
	// 3. Handle sample rates and channels properly

	// For testing, we'll just return the raw data
	return pcmData, nil
}
