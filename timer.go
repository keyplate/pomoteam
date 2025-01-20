package main

import (
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	//updates
	update  = "UPDATE"
	timeOut = "TIME_OUT"
	//commands
	start  = "START"
	stop   = "STOP"
	pause  = "PAUSE"
	resume = "RESUME"
)

type Timer struct {
	id          uuid.UUID
	ticker      *time.Ticker
	messages    chan TimerUpdate
	commands    chan TimerCommand
	duration    int
	currentTime int
	isActive    bool
	Done        chan bool
}

type TimerUpdate struct {
	timerId uuid.UUID
	name    string
	message string
}

type TimerCommand struct {
	name string
	arg  string
}

func New(id uuid.UUID) *Timer {
	ticker := time.NewTicker(1 * time.Second)
	messages := make(chan TimerUpdate, 1)
	done := make(chan bool, 1)
	timer := &Timer{
		id:          id,
		ticker:      ticker,
		duration:    0,
		currentTime: 0,
		messages:    messages,
		Done:        done,
	}
	go timer.listenCommands()
	return timer
}

func (t *Timer) Start(duration int) {
	if t.isActive {
		return
	}
	t.duration = duration
	t.currentTime = 0

	countdown := func() {
        defer t.ticker.Stop()

		for t.currentTime < t.duration {
			select {
			case <-t.ticker.C:
				t.currentTime++
				t.messages <- TimerUpdate{timerId: t.id, name: update, message: strconv.Itoa(t.currentTime)}
			case <-t.Done:
				return
			}
		}
		t.messages <- TimerUpdate{timerId: t.id, name: timeOut}
	}
	go countdown()
}

func (t *Timer) Pause() {
	t.ticker.Stop()
}

func (t *Timer) Resume() {
	t.ticker.Reset(1 * time.Second)
}

func (t *Timer) Subscribe() (chan TimerUpdate, chan bool) {
	return t.messages, t.Done
}

func (t *Timer) listenCommands() {
	for {
		select {
		case command := <-t.commands:
			err := t.parseCommand(command)
			log.Printf("ERROR: %v", err)
		case <-t.Done:
            return
		}
	}
}

func (t *Timer) parseCommand(command TimerCommand) error {
	switch command.name {
	case start:
		duration, err := strconv.Atoi(command.arg)
		if err != nil {
			return errors.New("Can not parse timer duration")
		}
		t.Start(duration)
	case pause:
		t.Pause()
	case resume:
		t.Resume()
	default:
		return errors.New("Unknown command")
	}
	return nil
}
