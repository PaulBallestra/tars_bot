package voice

import (
	"encoding/binary"
	"log"
	"sync"
	"time"

	"github.com/hraban/opus"
)

type AudioSender struct {
	Connection *VoiceConnection
	Queue      chan string
	Mutex      sync.Mutex
	Encoder    *opus.Encoder
}

func NewAudioSender(vc *VoiceConnection) (*AudioSender, error) {
	// Initialize Opus encoder with proper settings for Discord
	encoder, err := opus.NewEncoder(48000, 2, opus.AppVoIP)
	if err != nil {
		return nil, err
	}

	// Set bitrate to 64kbps which is good for voice
	encoder.SetBitrate(64000)

	return &AudioSender{
		Connection: vc,
		Queue:      make(chan string, 10),
		Encoder:    encoder,
	}, nil
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
	audioData, err := as.Connection.Agent.TTS.Generate(as.Connection.Context, text)
	if err != nil {
		log.Printf("Error generating TTS: %v", err)
		return
	}

	// Convert to Opus
	opusData, err := as.encodeToOpus(audioData)
	if err != nil {
		log.Printf("Error encoding to Opus: %v", err)
		return
	}

	// Send audio in chunks
	chunkSize := 960 // 20ms chunks at 48kHz
	for i := 0; i < len(opusData); i += chunkSize {
		end := i + chunkSize
		if end > len(opusData) {
			end = len(opusData)
		}

		chunk := opusData[i:end]
		as.Connection.VoiceConnection.Speaking(true)
		as.Connection.VoiceConnection.OpusSend <- chunk
		time.Sleep(20 * time.Millisecond) // Simulate real-time playback
	}

	as.Connection.VoiceConnection.Speaking(false)
}

func (as *AudioSender) encodeToOpus(pcmData []byte) ([]byte, error) {
	// Convert bytes to int16 samples
	samples := make([]int16, len(pcmData)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(pcmData[i*2:]))
	}

	// Create buffer for Opus data
	opusData := make([]byte, len(samples)*2) // Enough space for encoded data

	// Encode to Opus
	frameSize := len(samples) / 2 // 20ms frame size
	encoded, err := as.Encoder.Encode(samples[:frameSize], opusData)
	if err != nil {
		return nil, err
	}

	return opusData[:encoded], nil
}
