Project Structure : 

tars-bot/
├── cmd/
│   └── bot/
│       └── main.go          # Entry point
├── internal/
│   ├── ai/
│   │   ├── agent.go         # AI agent logic
│   │   ├── memory.go        # RAG memory implementation
│   │   └── openai/
│   │       ├── tts.go       # Text-to-speech
│   │       └── stt.go        # Speech-to-text
│   ├── discord/
│   │   ├── bot.go           # Discord bot core
│   │   ├── commands.go      # Discord bot Command Registration
│   │   └── handlers.go      # Message handlers
│   └── config/
│       └── config.go        # Config Implementation
├── pkg/
│   ├── utils/               # Utility functions
│   └── models/              # Data models
├── go.mod
├── go.sum
└── config.yaml              # Configuration file