package timer

import (
	"errors"
	"github.com/google/uuid"
	"sync"
)

type HubService struct {
	hubPool map[uuid.UUID]*hub
	mu      sync.RWMutex
}

func NewHubService() *HubService {
	return &HubService{
		hubPool: make(map[uuid.UUID]*hub),
	}
}

func (h *HubService) create() uuid.UUID {
	h.mu.Lock()
	defer h.mu.Unlock()

	timer := NewTimer()
	hub := newHub(timer)
	h.hubPool[hub.id] = hub
	go hub.run()
	return hub.id
}

func (h *HubService) delete(id uuid.UUID) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	hub, ok := h.hubPool[id]
	if !ok {
		return errors.New("hub not found")
	}
	close(hub.done)
	delete(h.hubPool, id)
	return nil
}

func (h *HubService) get(id uuid.UUID) (*hub, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hub, ok := h.hubPool[id]
	if !ok {
		return nil, errors.New("hub not found")
	}
	return hub, nil
}
