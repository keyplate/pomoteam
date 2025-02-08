package timer

import (
	"errors"
	"log"
	"strconv"
	"time"
)

const (
	//updates
	timeLeft         = "TIME_LEFT"
	timeOut          = "TIME_OUT"
	started          = "STARTED"
	stopped          = "STOPPED"
	durationAdjusted = "DURATION_ADJUSTED"
	closed           = "CLOSED"
	resumed          = "RESUMED"
	//commands
	start  = "START"
	stop   = "STOP"
	resume = "RESUME"
)

type timer struct {
	ticker   *time.Ticker
	updates  chan timerUpdate
	commands chan timerCommand
	duration int
	timeLeft int
	isActive bool
	done     chan bool
}

type timerUpdate struct {
	Name string `json:"name"`
	Arg  string `json:"arg,omitempty"`
}

type timerCommand struct {
	Name string `json:"name"`
	Arg  string `json:"arg,omitempty"`
}

func NewTimer() *timer {
	timer := &timer{
		ticker:   time.NewTicker(1 * time.Second),
		duration: 0,
		timeLeft: 0,
		updates:  make(chan timerUpdate),
		commands: make(chan timerCommand),
		done:     make(chan bool),
	}

	go timer.listenCommands()
	return timer
}

func (t *timer) start(duration int) {
	if t.isActive {
		return
	}

	t.duration = duration
	t.timeLeft = duration

	countdown := func() {
		defer func() {
			t.ticker.Stop()
			t.isActive = false
		}()

		t.isActive = true
		t.ticker.Reset(1 * time.Second)
		t.updates <- timerUpdate{Name: started}

		for t.timeLeft > 0 {
			select {
			case <-t.ticker.C:
				t.timeLeft--
				t.updates <- timerUpdate{Name: timeLeft, Arg: strconv.Itoa(t.timeLeft)}
			case <-t.done:
				return
			}
		}
		t.updates <- timerUpdate{Name: timeOut}
	}
	go countdown()
}

func (t *timer) stop() {
	t.ticker.Stop()
	t.updates <- timerUpdate{Name: stopped}
}

func (t *timer) resume() {
	t.ticker.Reset(1 * time.Second)
	t.updates <- timerUpdate{Name: resumed}
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
	case stop:
		t.stop()
	case resume:
		t.resume()
	default:
		return errors.New("timer parseCommand: unknown command")
	}
	return nil
}

func (t *timer) Close() {
	t.updates <- timerUpdate{Name: closed}
	close(t.done)
	close(t.updates)
	close(t.commands)
	t.ticker.Stop()
}
