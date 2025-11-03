package voice

import (
	"bytes"
	"encoding/binary"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hraban/opus"
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

	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in AudioReceiver: %v", r)
			log.Printf("Stack trace: %s", debug.Stack())
		}
	}()

	opusBuffer := make([]byte, 0, 5*48000*2) // Buffer for 5 seconds at 48kHz, stereo
	opusChan := make(chan *discordgo.Packet, 100)
	lastProcessed := time.Now()
	receivedCount := 0
	reconnectAttempts := 0

	// SSRC to User ID mapping
	ssrcToUserID := make(map[uint32]string)

	// Function to setup the audio receiver
	setupReceiver := func() {
		if ar.Connection.VoiceConnection == nil {
			log.Println("ERROR: Voice connection is nil during setupReceiver")
			return
		}

		// Enable receiving Opus packets
		log.Println("Setting up OpusRecv channel")

		// DO NOT close the existing channel, as it's managed by discordgo
		// Just set our new channel
		ar.Connection.VoiceConnection.OpusRecv = opusChan
		log.Printf("OpusRecv channel set: %p", opusChan)

		// Ensure speaking is enabled
		log.Println("Setting Speaking state to true")
		err := ar.Connection.VoiceConnection.Speaking(true)
		if err != nil {
			log.Printf("Error setting speaking state: %v", err)
		}

		// Check if voice is ready
		log.Printf("Voice connection ready state: %v", ar.Connection.VoiceConnection.Ready)

		// Attempt to set speaking one more time to ensure it's enabled
		time.Sleep(500 * time.Millisecond)
		ar.Connection.VoiceConnection.Speaking(true)
	}

	// Initial setup
	setupReceiver()

	// Ensure Speaking state is set to false on exit
	defer func() {
		if ar.Connection.VoiceConnection != nil {
			ar.Connection.VoiceConnection.Speaking(false)
		}
	}()

	log.Println("Entering audio reception loop")
	tick := time.NewTicker(10 * time.Second)
	debug := time.NewTicker(1 * time.Second)
	pingTick := time.NewTicker(30 * time.Second) // Send a ping every 30 seconds to keep connection alive
	resetTick := time.NewTicker(2 * time.Minute) // Completely reset the receiver channel periodically
	defer tick.Stop()
	defer debug.Stop()
	defer pingTick.Stop()
	defer resetTick.Stop()

	// Log voice status periodically
	statusTicker := time.NewTicker(15 * time.Second)
	defer statusTicker.Stop()

	log.Println("Starting user speech detection...")
	// Make sure we log which users are present in the voice channel
	if ar.Connection.VoiceConnection != nil && ar.Connection.VoiceConnection.Ready {
		guild, err := ar.Connection.Session.State.Guild(ar.Connection.GuildID)
		if err == nil {
			log.Printf("Guild: %s has %d voice states", guild.Name, len(guild.VoiceStates))
			for _, vs := range guild.VoiceStates {
				if vs.ChannelID == ar.Connection.ChannelID {
					user, err := ar.Connection.Session.User(vs.UserID)
					if err == nil {
						log.Printf("User in voice channel: %s (ID: %s, SelfMute: %v, SelfDeaf: %v, Mute: %v, Deaf: %v)",
							user.Username, user.ID, vs.SelfMute, vs.SelfDeaf, vs.Mute, vs.Deaf)
					}
				}
			}
		}
	}

	for {
		select {
		case <-ar.Connection.Context.Done():
			log.Println("Context done, exiting audio receiver")
			return
		case <-tick.C:
			log.Printf("Audio receiver still active, waiting for packets... (received %d so far)", receivedCount)
		case <-resetTick.C:
			// Completely reset the receiver channel periodically
			log.Println("Performing periodic reset of audio receiver...")
			if ar.Connection.VoiceConnection != nil && ar.Connection.VoiceConnection.Ready {
				// Force re-create the OpusRecv channel
				setupReceiver()

				// Perform additional diagnostics
				ar.debugVoiceConnection()

				log.Println("Audio receiver reset completed")
			}
		case <-pingTick.C:
			// Send a ping to keep the connection alive
			if ar.Connection.VoiceConnection != nil && ar.Connection.VoiceConnection.Ready {
				log.Println("Sending voice ping to keep connection alive")
				ar.Connection.VoiceConnection.Speaking(true)
				time.Sleep(100 * time.Millisecond)
				ar.Connection.VoiceConnection.Speaking(false)
				time.Sleep(100 * time.Millisecond)
				ar.Connection.VoiceConnection.Speaking(true)
			}
		case <-statusTicker.C:
			// Log detailed voice connection status
			if ar.Connection.VoiceConnection != nil {
				log.Printf("Voice connection status - Ready: %v, Speaking: %v, OpusRecv: %p",
					ar.Connection.VoiceConnection.Ready,
					ar.Connection.VoiceConnection.Speaking,
					ar.Connection.VoiceConnection.OpusRecv)

				// Force refresh the voice connection
				log.Println("Refreshing voice connection...")
				ar.Connection.VoiceConnection.Speaking(false)
				time.Sleep(100 * time.Millisecond)
				ar.Connection.VoiceConnection.Speaking(true)

				// Log current users in the channel
				guild, err := ar.Connection.Session.State.Guild(ar.Connection.GuildID)
				if err == nil {
					log.Printf("Users currently in voice channel:")
					for _, vs := range guild.VoiceStates {
						if vs.ChannelID == ar.Connection.ChannelID {
							user, err := ar.Connection.Session.User(vs.UserID)
							if err == nil {
								log.Printf("- %s (ID: %s, SelfMute: %v, SelfDeaf: %v)",
									user.Username, user.ID, vs.SelfMute, vs.SelfDeaf)
							}
						}
					}
				}
			}
		case <-debug.C:
			// Check if the voice connection is still active
			if ar.Connection.VoiceConnection == nil {
				log.Println("Voice connection is nil")
				// Try to reconnect
				if reconnectAttempts < 5 {
					reconnectAttempts++
					log.Printf("Attempting to reconnect (attempt %d/5)...", reconnectAttempts)
					err := ar.Connection.Connect()
					if err != nil {
						log.Printf("Failed to reconnect: %v", err)
					} else {
						reconnectAttempts = 0
						setupReceiver()
					}
				}
			} else {
				ready := ar.Connection.VoiceConnection.Ready
				log.Printf("Voice connection ready: %v", ready)

				// Print more detailed info about the speaking state
				speakState := ar.Connection.VoiceConnection.Speaking
				log.Printf("Voice connection speaking state: %v", speakState)

				// Check if OpusRecv channel is still valid
				log.Printf("OpusRecv channel: %p (nil: %v)",
					ar.Connection.VoiceConnection.OpusRecv,
					ar.Connection.VoiceConnection.OpusRecv == nil)

				// Verify the voice channel has users
				guild, err := ar.Connection.Session.State.Guild(ar.Connection.GuildID)
				if err == nil {
					usersInChannel := 0
					for _, vs := range guild.VoiceStates {
						if vs.ChannelID == ar.Connection.ChannelID {
							usersInChannel++
						}
					}
					log.Printf("Users in voice channel: %d", usersInChannel)
				}

				// If connection is not ready, try to re-establish
				if !ready && reconnectAttempts < 5 {
					reconnectAttempts++
					log.Printf("Voice connection not ready, attempting to reconnect (attempt %d/5)...", reconnectAttempts)
					setupReceiver()
				}
			}
		case packet := <-opusChan:
			if packet == nil {
				log.Println("Received nil packet")
				continue
			}
			receivedCount++

			// Log detailed packet information
			log.Printf("PACKET RECEIVED: SSRC=%d, Type=%d, Sequence=%d, Timestamp=%d, Size=%d bytes",
				packet.SSRC, packet.Type, packet.Sequence, packet.Timestamp, len(packet.Opus))

			// Get the user who is speaking
			userID, ok := ssrcToUserID[packet.SSRC]
			if !ok {
				// If we don't have a mapping yet, try to find the user from voice states
				guild, err := ar.Connection.Session.State.Guild(ar.Connection.GuildID)
				if err == nil {
					for _, vs := range guild.VoiceStates {
						if vs.ChannelID == ar.Connection.ChannelID {
							// We don't have direct access to SSRC from voice states
							// This is a simplification - in a real implementation
							// you would need a proper way to map SSRC to users
							ssrcToUserID[packet.SSRC] = vs.UserID
							userID = vs.UserID
							break
						}
					}
				}
			}

			if userID != "" {
				user, err := ar.Connection.Session.User(userID)
				if err != nil {
					log.Printf("Could not identify user for SSRC %d (UserID: %s): %v", packet.SSRC, userID, err)
				} else {
					log.Printf("Received audio from user: %s (ID: %s)", user.Username, user.ID)
				}
			} else {
				log.Printf("Could not map SSRC %d to a user", packet.SSRC)
			}

			log.Printf("Received Opus packet with %d bytes from SSRC: %d", len(packet.Opus), packet.SSRC)

			// Debug first few bytes of the packet
			if len(packet.Opus) > 0 {
				previewSize := 8
				if len(packet.Opus) < previewSize {
					previewSize = len(packet.Opus)
				}
				log.Printf("Packet data preview: %v", packet.Opus[:previewSize])
			}

			ar.Mutex.Lock()
			opusBuffer = append(opusBuffer, packet.Opus...)
			ar.Mutex.Unlock()

			// Process every 5 seconds or when buffer is large enough
			if time.Since(lastProcessed) >= 5*time.Second || len(opusBuffer) >= 5*48000*2 {
				ar.Mutex.Lock()
				if len(opusBuffer) > 0 {
					log.Printf("Processing audio chunk of %d bytes", len(opusBuffer))
					ar.processAudioChunk(opusBuffer)
					opusBuffer = opusBuffer[:0] // Reset buffer
				} else {
					log.Println("Buffer is empty, nothing to process")
				}
				ar.Mutex.Unlock()
				lastProcessed = time.Now()
			}
		}
	}
}

func (ar *AudioReceiver) processAudioChunk(opusData []byte) {
	// Convert Opus to PCM for STT compatibility
	pcmData, err := opusToWav(opusData)
	if err != nil {
		log.Printf("Error converting Opus to WAV: %v", err)
		return
	}

	// Send to STT
	text, err := ar.Connection.Agent.STT.Transcribe(ar.Connection.Context, pcmData)
	if err != nil {
		log.Printf("Error transcribing audio: %v", err)
		return
	}

	if text == "" {
		log.Println("Empty transcription received")
		return
	}

	log.Printf("Transcribed text: %s", text)

	// Process with AI agent - add userID parameter (using empty string as fallback)
	response, err := ar.Connection.Agent.ProcessMessage(ar.Connection.Context, text, "")
	if err != nil {
		log.Printf("Error processing message: %v", err)
		return
	}

	log.Printf("AI response: %s", response)

	// Send TTS response
	ar.Connection.AudioSender.QueueResponse(response)
}

// opusToWav converts Opus data to WAV format, which is more widely supported by STT services
func opusToWav(opusData []byte) ([]byte, error) {
	// Initialize Opus decoder
	decoder, err := opus.NewDecoder(48000, 2)
	if err != nil {
		return nil, err
	}

	// Decode Opus data to PCM
	pcmData := make([]int16, 48000*2*5) // 5 seconds buffer at 48kHz stereo
	samplesDecoded, err := decoder.Decode(opusData, pcmData)
	if err != nil {
		return nil, err
	}

	pcmData = pcmData[:samplesDecoded]

	// Create WAV header
	var wavHeader struct {
		ChunkID       [4]byte // "RIFF"
		ChunkSize     uint32  // 4 + (8 + SubChunk1Size) + (8 + SubChunk2Size)
		Format        [4]byte // "WAVE"
		SubChunk1ID   [4]byte // "fmt "
		SubChunk1Size uint32  // 16 for PCM
		AudioFormat   uint16  // 1 for PCM
		NumChannels   uint16  // 2 for stereo
		SampleRate    uint32  // 48000
		ByteRate      uint32  // SampleRate * NumChannels * BitsPerSample/8
		BlockAlign    uint16  // NumChannels * BitsPerSample/8
		BitsPerSample uint16  // 16
		SubChunk2ID   [4]byte // "data"
		SubChunk2Size uint32  // NumSamples * NumChannels * BitsPerSample/8
	}

	// Fill in WAV header
	copy(wavHeader.ChunkID[:], []byte("RIFF"))
	copy(wavHeader.Format[:], []byte("WAVE"))
	copy(wavHeader.SubChunk1ID[:], []byte("fmt "))
	copy(wavHeader.SubChunk2ID[:], []byte("data"))

	wavHeader.SubChunk1Size = 16
	wavHeader.AudioFormat = 1
	wavHeader.NumChannels = 2
	wavHeader.SampleRate = 48000
	wavHeader.BitsPerSample = 16
	wavHeader.ByteRate = wavHeader.SampleRate * uint32(wavHeader.NumChannels) * uint32(wavHeader.BitsPerSample) / 8
	wavHeader.BlockAlign = wavHeader.NumChannels * wavHeader.BitsPerSample / 8
	wavHeader.SubChunk2Size = uint32(len(pcmData) * 2)
	wavHeader.ChunkSize = 4 + (8 + wavHeader.SubChunk1Size) + (8 + wavHeader.SubChunk2Size)

	// Create WAV file
	var wavBuffer bytes.Buffer
	binary.Write(&wavBuffer, binary.LittleEndian, wavHeader)

	// Write PCM data
	binary.Write(&wavBuffer, binary.LittleEndian, pcmData)

	wavBytes := wavBuffer.Bytes()
	log.Printf("Created WAV data of size %d bytes", len(wavBytes))

	return wavBytes, nil
}

// debugVoiceConnection provides additional diagnostics for voice connection issues
func (ar *AudioReceiver) debugVoiceConnection() {
	if ar.Connection.VoiceConnection == nil {
		log.Println("Cannot debug - voice connection is nil")
		return
	}

	// Log the current guild voice states
	guild, err := ar.Connection.Session.State.Guild(ar.Connection.GuildID)
	if err != nil {
		log.Printf("Error getting guild: %v", err)
		return
	}

	log.Printf("Voice states for guild %s (%s):", guild.Name, guild.ID)
	for _, vs := range guild.VoiceStates {
		username := vs.UserID
		user, err := ar.Connection.Session.User(vs.UserID)
		if err == nil {
			username = user.Username
		}

		log.Printf("User %s (ID: %s) in channel %s (SelfMute: %v, SelfDeaf: %v, Suppress: %v)",
			username, vs.UserID, vs.ChannelID, vs.SelfMute, vs.SelfDeaf, vs.Suppress)
	}

	// Verify our bot's permissions in the channel
	channel, err := ar.Connection.Session.Channel(ar.Connection.ChannelID)
	if err == nil {
		log.Printf("Voice channel type: %d, name: %s", channel.Type, channel.Name)
	}

	// Check if we're in the correct voice channel
	inChannel := false
	for _, vs := range guild.VoiceStates {
		if vs.UserID == ar.Connection.Session.State.User.ID &&
			vs.ChannelID == ar.Connection.ChannelID {
			inChannel = true
			log.Printf("Bot confirmed in voice channel %s", vs.ChannelID)
			break
		}
	}

	if !inChannel {
		log.Printf("WARNING: Bot does not appear to be in the expected voice channel!")
	}

	// Log voice connection details
	log.Printf("Voice connection ready: %v", ar.Connection.VoiceConnection.Ready)
	log.Printf("Voice connection speaking: %v", ar.Connection.VoiceConnection.Speaking)
}
