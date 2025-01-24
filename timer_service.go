package main

import (
	"errors"

	"github.com/google/uuid"
)

type TimerService struct {
	timerPool map[uuid.UUID]*timer
}

func (t *TimerService) Create() uuid.UUID {
	timer := New()
	t.timerPool[timer.id] = timer
	return timer.id
}

func (t *TimerService) Delete(id uuid.UUID) error {
	timer, ok := t.timerPool[id]
	if !ok {
		return errors.New("There is no timer with this id")
	}

	timer.close()
	return nil
}

func (t *TimerService) Get(id uuid.UUID) (*timer, error) {
	timer, ok := t.timerPool[id]
	if !ok {
		return nil, errors.New("there is no timer with this id")
	}
	return timer, nil
}
