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
	currentTime = "CURRENT_TIME"
	timeOut     = "TIME_OUT"
	//commands
	start  = "START"
	stop   = "STOP"
	pause  = "PAUSE"
	resume = "RESUME"
)

type timer struct {
	id          uuid.UUID
	ticker      *time.Ticker
	messages    chan timerUpdate
	commands    chan timerCommand
	duration    int
	currentTime int
	isActive    bool
	done        chan bool
}

type timerUpdate struct {
	timerId uuid.UUID
	name    string
	message string
}

type timerCommand struct {
	name string
	arg  string
}

func New(id uuid.UUID) *timer {
	ticker := time.NewTicker(1 * time.Second)
	messages := make(chan timerUpdate, 1)
	commands := make(chan timerCommand, 1)
	done := make(chan bool, 1)

	timer := &timer{
		id:          id,
		ticker:      ticker,
		duration:    0,
		currentTime: 0,
		messages:    messages,
		commands:    commands,
		done:        done,
	}

	go timer.listenCommands()
	return timer
}

func (t *timer) start(duration int) {
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
				t.messages <- timerUpdate{timerId: t.id, name: currentTime, message: strconv.Itoa(t.currentTime)}
			case <-t.done:
				return
			}
		}
		t.messages <- timerUpdate{timerId: t.id, name: timeOut}
	}
	go countdown()
}

func (t *timer) pause() {
	t.ticker.Stop()
}

func (t *timer) resume() {
	t.ticker.Reset(1 * time.Second)
}

func (t *timer) listenCommands() {
	for {
		select {
		case command := <-t.commands:
			err := t.parseCommand(command)
			if err != nil {
				log.Print(err)
			}
		case <-t.done:
			return
		}
	}
}

func (t *timer) parseCommand(command timerCommand) error {
	switch command.name {
	case start:
		duration, err := strconv.Atoi(command.arg)
		if err != nil {
			return errors.New("timer parseCommand: can not parse timer duration")
		}
		t.start(duration)
	case pause:
		t.pause()
	case resume:
		t.resume()
	default:
		return errors.New("timer parseCommand: unknown command")
	}
	return nil
}

func (t *timer) close() {
    close(t.done)
    close(t.messages)
    close(t.commands)
    t.ticker.Stop()
}
