package main

import (
	"errors"
	"log"
	"strconv"
	"time"
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
	ticker      *time.Ticker
	updates     chan timerUpdate
	commands    chan timerCommand
	duration    int
	currentTime int
	isActive    bool
	done        chan bool
}

type timerUpdate struct {
	Name string `json:"name"`
	Arg  string `json:"arg"`
}

type timerCommand struct {
	Name string `json:"name"`
	Arg  string `json:"arg"`
}

func New() *timer {
	timer := &timer{
		ticker:      time.NewTicker(1 * time.Second),
		duration:    0,
		currentTime: 0,
		updates:     make(chan timerUpdate),
		commands:    make(chan timerCommand),
		done:        make(chan bool),
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
		defer func() {
			t.ticker.Stop()
			t.isActive = false
		}()

		t.isActive = true
		t.ticker.Reset(1 * time.Second)
		for t.currentTime < t.duration {
			select {
			case <-t.ticker.C:
				t.currentTime++
				t.updates <- timerUpdate{Name: currentTime, Arg: strconv.Itoa(t.currentTime)}
			case <-t.done:
				return
			}
		}
		t.updates <- timerUpdate{Name: timeOut}
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
	switch command.Name {
	case start:
		duration, err := strconv.Atoi(command.Arg)
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
	close(t.updates)
	close(t.commands)
	t.ticker.Stop()
}
