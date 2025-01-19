package main

import (
	"errors"

	"github.com/google/uuid"
)

type TimerService struct {
    timerPool map[uuid.UUID]Timer
}

func (t *TimerService)Create() uuid.UUID {
    id := uuid.New()
    timer := New(id)
    t.timerPool[id] = *timer
    return id
}

func (t *TimerService)Delete(id uuid.UUID) error {
    timer, ok := t.timerPool[id]
    if !ok {
        return errors.New("There is no timer with this id")
    }

    timer.Delete()
    return nil
}
