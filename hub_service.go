package main

import (
	"errors"

	"github.com/google/uuid"
)

type HubService struct {
	hubPool map[uuid.UUID]*hub
}

func (h *HubService) Create() uuid.UUID {
	timer := New()
	hub := newHub(timer)
	h.hubPool[hub.id] = hub
	return hub.id
}

func (h *HubService) Delete(id uuid.UUID) error {
	hub, ok := h.hubPool[id]
	if !ok {
		return errors.New("There is no hub with this id")
	}

	close(hub.done)
	return nil
}

func (h *HubService) Get(id uuid.UUID) (*hub, error) {
	hub, ok := h.hubPool[id]
	if !ok {
		return nil, errors.New("there is no timer with this id")
	}
	return hub, nil
}
