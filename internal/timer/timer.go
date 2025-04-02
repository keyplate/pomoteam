// This package represents a concurrent timer for the flow-time techenque (which is a bit more agile version of pomodoro).
// Timer is operated by using commands. Timer publishes updates notifying users about its state changes.
package timer

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	// Session types
	sessionFocus = "FOCUS"
	sessionBreak = "BREAK"
)

// Timer represents a pomodoro-style timer that alternates between focus and break sessions
type timer struct {
	// Channels for communication
	ticker  *time.Ticker
	updates chan Update

	// Session control
	sessionDone chan struct{}

	// Lifecycle control
	ctx context.Context

	// Session configuration
	breakDuration int
	focusDuration int

	// Current state
	timeLeft       int64
	isRunning      bool
	isSessionEnded bool
	sessionType    string
}

func NewTimer(ctx context.Context, updates chan Update) *timer {
	breakDuration := 300
	focusDuration := 1500
	sessionType := sessionFocus

	timer := &timer{
		ticker:  time.NewTicker(1 * time.Second),
		updates: updates,
		ctx:     ctx,

		breakDuration:  breakDuration,
		focusDuration:  focusDuration,
		isRunning:      false,
		isSessionEnded: true,
		sessionType:    sessionType,
	}

	return timer
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

func (t *timer) reset() {
	if !t.isRunning || t.isSessionEnded {
		return
	}

	t.isRunning = false
	t.isSessionEnded = true
	t.ticker.Stop()

	t.sendUpdate(timerReset, map[string]string{
		"isRunning":      strconv.FormatBool(t.isRunning),
		"isSessionEnded": strconv.FormatBool(t.isSessionEnded),
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
		t.adjustTimeLeft(delta)
		return
	}
	t.adjustSessionDuration(delta)
}

func (t *timer) adjustTimeLeft(delta int) {
	if t.timeLeft+int64(delta) <= 0 {
		atomic.StoreInt64(&t.timeLeft, 0)
	} else {
		atomic.AddInt64(&t.timeLeft, int64(delta))
	}

	t.sendUpdate(durationAdjusted, map[string]string{
		"timeLeft":      strconv.Itoa(int(t.timeLeft)),
		"breakDuration": strconv.Itoa(t.breakDuration),
		"focusDuration": strconv.Itoa(t.focusDuration),
	})
}

func (t *timer) adjustSessionDuration(delta int) {
	var session *int
	if t.sessionType == sessionBreak {
		session = &t.breakDuration
	} else {
		session = &t.focusDuration
	}

	if *session+delta <= 0 {
		*session = 300
	} else {
		*session += delta
	}

	t.sendUpdate(durationAdjusted, map[string]string{
		"timeLeft":      strconv.Itoa(int(t.timeLeft)),
		"breakDuration": strconv.Itoa(t.breakDuration),
		"focusDuration": strconv.Itoa(t.focusDuration),
	})
}

func (t *timer) sendUpdate(name string, args map[string]string) {
	slog.Debug(fmt.Sprintf("timer: update pending - name: %s, args: %v", name, args))

	select {
	case <-t.ctx.Done():
		return
	case t.updates <- Update{Name: name, Args: args}:
		slog.Debug(fmt.Sprintf("timer: update sent - name: %s, args: %v", name, args))
	}
}

func (t *timer) close() {
	// Stop the timer
	t.ticker.Stop()

	// Stop current session if running
	if t.sessionDone != nil {
		close(t.sessionDone)
	}

	// Close channels in correct order
	close(t.updates)
}
