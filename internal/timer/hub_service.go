package timer

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

type HubService struct {
	hubPool map[uuid.UUID]*hub
	mu      sync.RWMutex
}

func NewHubService() *HubService {
	hs := &HubService{
		hubPool: make(map[uuid.UUID]*hub),
	}
	return hs
}

func (h *HubService) create() uuid.UUID {
	h.mu.Lock()
	defer h.mu.Unlock()

	//todo: allow timer creation from user
	hub := newHub(h.unregister)
	h.hubPool[hub.id] = hub
	go hub.run()
	return hub.id
}

func (h *HubService) unregister(id uuid.UUID) {
	err := h.delete(id)
	if err != nil {
		slog.Warn(fmt.Sprintf("hubService: %v", err))
	}
}

func (h *HubService) delete(id uuid.UUID) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	_, ok := h.hubPool[id]
	if !ok {
		return errors.New("hub not found")
	}
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
