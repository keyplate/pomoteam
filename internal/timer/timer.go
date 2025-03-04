// This package represents a concurrent timer for the flow-time techenque (which is a bit more agile version of pomodoro).
// Timer is operated by using commands. Timer publishes updates notifying users about its state changes.
package timer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	// Update event types
	timeUpdate       = "TIME_UPDATE"
	timeOut          = "TIME_OUT"
	started          = "STARTED"
	stopped          = "STOPPED"
	durationAdjusted = "DURATION_ADJUSTED"
	closed           = "CLOSED"
	resumed          = "RESUMED"
	paused           = "PAUSED"
	sessionUpdate    = "SESSION_UPDATE"
)

const (
	// Command types
	start  = "START"
	stop   = "STOP"
	pause  = "PAUSE"
	resume = "RESUME"
	adjust = "ADJUST"
)

const (
	// Session types
	sessionFocus = "FOCUS"
	sessionBreak = "BREAK"
)

type timerCommand struct {
	Name string `json:"name"`
	Arg  string `json:"args"`
}

type timerUpdate struct {
	Name string            `json:"name"`
	Args map[string]string `json:"args"`
}

// Timer represents a pomodoro-style timer that alternates between focus and break sessions
type timer struct {
	// Channels for communication
	ticker   *time.Ticker
	updates  chan timerUpdate
	commands chan timerCommand

	// Session control
	sessionDone chan struct{}

	// Lifecycle control
	cancel context.CancelFunc
	ctx    context.Context

	// Session configuration
	breakDuration int
	focusDuration int

	// Current state
	timeLeft       int64
	isRunning      bool
	isSessionEnded bool
	sessionType    string
}

func NewTimer(ctx context.Context, breakDuration, focusDuration int, sessionType string) *timer {
	ctx, cancel := context.WithCancel(ctx)

	timer := &timer{
		ticker:   time.NewTicker(1 * time.Second),
		updates:  make(chan timerUpdate),
		commands: make(chan timerCommand),

		ctx:    ctx,
		cancel: cancel,

		breakDuration:  breakDuration,
		focusDuration:  focusDuration,
		isRunning:      false,
		isSessionEnded: true,
		sessionType:    sessionType,
	}

	go timer.listenCommands()
	return timer
}

func (t *timer) listenCommands() {
	for {
		select {
		case command := <-t.commands:
			err := t.parseCommand(command)
			if err != nil {
				log.Print(err)
			}
		case <-t.ctx.Done():
			return
		}
	}
}

func (t *timer) parseCommand(command timerCommand) error {
	switch command.Name {
	case start:
		t.start()
	case pause:
		t.pause()
	case resume:
		t.resume()
	case adjust:
		duration, err := strconv.Atoi(command.Arg)
		if err != nil {
			return errors.New("timer parseCommand: can not parse timer duration")
		}
		t.adjust(duration)
	default:
		return fmt.Errorf("unknown command: %v", command)
	}
	return nil
}

func (t *timer) start() {
	if t.isRunning {
		return
	}

	// Stop any existing countdown
	if t.sessionDone != nil {
		close(t.sessionDone)
	}
	t.sessionDone = make(chan struct{})

	// Set initial duration based on session type
	t.timeLeft = int64(t.getDurationForSession())

	t.ticker = time.NewTicker(1 * time.Second)
	go t.countdown()
}

func (t *timer) getDurationForSession() int {
	if t.sessionType == sessionBreak {
		return t.breakDuration
	}
	return t.focusDuration
}

func (t *timer) countdown() {
	defer t.ticker.Stop()

	t.isRunning = true
	t.isSessionEnded = false
	t.sendUpdate(started, map[string]string{
		"isRunning":      strconv.FormatBool(t.isRunning),
		"isSessionEnded": strconv.FormatBool(t.isSessionEnded),
		"timeLeft":       strconv.Itoa(int(t.timeLeft)),
	})

	for t.timeLeft > 0 {
		select {
		case <-t.ticker.C:
			t.tick()
		case <-t.sessionDone:
			return
		case <-t.ctx.Done():
			return
		}
	}

	t.switchSession()
	t.isRunning = false
	t.isSessionEnded = true
	t.sendUpdate(timeOut, map[string]string{
		"isRunning":      strconv.FormatBool(t.isRunning),
		"isSessionEnded": strconv.FormatBool(t.isSessionEnded),
	})
}

// tick represents a logical tick of the timer and an actual passage of 1s.
func (t *timer) tick() {
	atomic.AddInt64(&t.timeLeft, -1)

	timeLeft := int(t.timeLeft)
	duration := t.getDurationForSession()

	t.sendUpdate(timeUpdate, map[string]string{
		"timeLeft": strconv.Itoa(timeLeft),
		"duration": strconv.Itoa(duration),
	})
}

func (t *timer) pause() {
	if !t.isRunning || t.isSessionEnded {
		return
	}
	t.isRunning = false
	t.ticker.Stop()

	t.sendUpdate(paused, map[string]string{
		"isRunning": strconv.FormatBool(t.isRunning),
	})
}

func (t *timer) resume() {
	if t.isRunning || t.isSessionEnded {
		return
	}
	t.isRunning = true
	t.ticker.Reset(1 * time.Second)

	t.sendUpdate(resumed, map[string]string{
		"isRunning": strconv.FormatBool(t.isRunning),
	})
}

func (t *timer) switchSession() {
	if t.sessionType == sessionBreak {
		t.sessionType = sessionFocus
	} else {
		t.sessionType = sessionBreak
	}

	t.sendUpdate(sessionUpdate, map[string]string{
		"sessionType": t.sessionType,
	})
}

// adjust alters timer duratin by given delta.
func (t *timer) adjust(delta int) {
	if t.isRunning || !t.isSessionEnded {
		atomic.AddInt64(&t.timeLeft, int64(delta))
		t.sendUpdate(durationAdjusted, map[string]string{
			"timeLeft": strconv.Itoa(int(t.timeLeft)),
		})
		return
	}

	if t.sessionType == sessionBreak {
		t.breakDuration += delta
	} else {
		t.focusDuration += delta
	}

	t.sendUpdate(durationAdjusted, map[string]string{
		"timeLeft":      strconv.Itoa(int(t.timeLeft)),
		"breakDuration": strconv.Itoa(t.breakDuration),
		"focusDuration": strconv.Itoa(t.focusDuration),
	})
}

func (t *timer) sendUpdate(name string, args map[string]string) {
	t.updates <- timerUpdate{
		Name: name,
		Args: args,
	}
}

func (t *timer) Close() {
	t.sendUpdate(closed, nil)

	// Cancel context to stop all goroutines
	t.cancel()

	// Stop the timer
	t.ticker.Stop()

	// Stop current session if running
	if t.sessionDone != nil {
		close(t.sessionDone)
	}

	// Close channels in correct order
	close(t.updates)
	close(t.commands)
}
