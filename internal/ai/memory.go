package ai

import (
	"sync"
	"tars-bot/pkg/models"
)

type Memory struct {
	mu      sync.Mutex
	storage map[string][]models.Interaction // userID -> interactions
}

func NewMemory() *Memory {
	return &Memory{
		storage: make(map[string][]models.Interaction),
	}
}

func (m *Memory) Store(userID, input, response string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Limit to last 10 interactions to prevent memory bloat
	if len(m.storage[userID]) >= 10 {
		m.storage[userID] = append(m.storage[userID][1:], models.Interaction{
			Input:    input,
			Response: response,
		})
	} else {
		m.storage[userID] = append(m.storage[userID], models.Interaction{
			Input:    input,
			Response: response,
		})
	}
}

func (m *Memory) Retrieve(userID, currentMessage string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	interactions := m.storage[userID]
	if len(interactions) == 0 {
		return "No previous context. Current message: " + currentMessage
	}

	// Format the context
	context := "Previous conversation context:\n"
	for _, i := range interactions {
		context += "User: " + i.Input + "\nAI: " + i.Response + "\n"
	}

	return context + "Current message: " + currentMessage
}
