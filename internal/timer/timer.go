// This package represents a concurrent timer for the flow-time technique (which is a bit more agile version of pomodoro).
// Timer is operated by using commands. Timer publishes updates notifying users about its state changes.
package timer

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
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
	ticker     *time.Ticker
	updates    chan Update
	commands   chan Command
	ctx        context.Context
	mu         sync.Mutex
	stateCache timerState
	timerState
}

type timerState struct {
	// Session configuration
	breakDuration int
	focusDuration int

	// Current state
	timeLeft       int
	isRunning      bool
	isSessionEnded bool
	sessionType    string
}

func NewTimer(ctx context.Context, updates chan Update) *timer {
	breakDuration := 300
	focusDuration := 1500
	sessionType := sessionFocus
	ticker := time.NewTicker(1 * time.Second)
	ticker.Stop()

	timer := &timer{
		ticker:   ticker,
		updates:  updates,
		commands: make(chan Command),
		ctx:      ctx,

		timerState: timerState{
			breakDuration:  breakDuration,
			focusDuration:  focusDuration,
			isRunning:      false,
			isSessionEnded: true,
			sessionType:    sessionType,
		},
	}
	timer.stateCache = timer.timerState

	return timer
}

func (t *timer) run() {
	for {
		select {
		case cmd := <-t.commands:
			slog.Debug(fmt.Sprintf("timer - executing command: %v", cmd))
			err := t.executeCommand(cmd)
			if err != nil {
				slog.Debug(fmt.Sprintf("timer - command executeion error: %v", err))
			}
		case <-t.ticker.C:
			t.tick()
		case <-t.ctx.Done():
			t.close()
			return
		default:
			t.mu.Lock()
			t.stateCache = t.timerState
			t.mu.Unlock()
		}

	}
}

func (t *timer) executeCommand(cmd Command) error {
	switch cmd.Name {
	case start:
		t.start()
	case pause:
		t.pause()
	case resume:
		t.resume()
	case reset:
		t.reset()
	case adjust:
		duration, err := strconv.Atoi(cmd.Args["duration"])
		if err != nil {
			return fmt.Errorf("can not parse timer duration")
		}
		t.adjust(duration)
	default:
		return fmt.Errorf("unknown command: %v", cmd)
	}
	return nil
}

func (t *timer) start() {
	if t.isRunning {
		return
	}

	// Set initial duration based on session type
	t.timeLeft = t.getDurationForSession()
	t.isRunning = true
	t.isSessionEnded = false
	t.sendUpdate(started, map[string]any{
		"isRunning":      t.isRunning,
		"isSessionEnded": t.isSessionEnded,
		"timeLeft":       t.timeLeft,
	})

	t.ticker.Reset(1 * time.Second)
}

func (t *timer) getDurationForSession() int {
	if t.sessionType == sessionBreak {
		return t.breakDuration
	}
	return t.focusDuration
}

// tick represents a logical tick of the timer and an actual passage of 1s.
func (t *timer) tick() {
	if t.timeLeft <= 0 {
		t.timeOut()
		return
	}

	t.timeLeft--
	duration := t.getDurationForSession()

	t.sendUpdate(timeUpdate, map[string]any{
		"timeLeft": t.timeLeft,
		"duration": duration,
	})
}

func (t *timer) timeOut() {
	t.ticker.Stop()
	t.isRunning = false
	t.isSessionEnded = true
	t.switchSession()

	t.sendUpdate(timeOut, map[string]any{
		"isRunning":      t.isRunning,
		"isSessionEnded": t.isSessionEnded,
	})
}

func (t *timer) pause() {
	if !t.isRunning || t.isSessionEnded {
		return
	}
	t.isRunning = false
	t.ticker.Stop()

	t.sendUpdate(paused, map[string]any{
		"isRunning": t.isRunning,
	})
}

func (t *timer) reset() {
	if !t.isRunning || t.isSessionEnded {
		return
	}

	t.isRunning = false
	t.isSessionEnded = true
	t.ticker.Stop()

	t.sendUpdate(timerReset, map[string]any{
		"isRunning":      t.isRunning,
		"isSessionEnded": t.isSessionEnded,
	})
}

func (t *timer) resume() {
	if t.isRunning || t.isSessionEnded {
		return
	}

	t.isRunning = true
	t.ticker.Reset(1 * time.Second)

	t.sendUpdate(resumed, map[string]any{
		"isRunning": t.isRunning,
	})
}

func (t *timer) switchSession() {
	if t.sessionType == sessionBreak {
		t.sessionType = sessionFocus
	} else {
		t.sessionType = sessionBreak
	}

	t.sendUpdate(sessionUpdate, map[string]any{
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
	if t.timeLeft+delta <= 0 {
		t.timeLeft = 0
	} else {
		t.timeLeft += delta
	}

	t.sendUpdate(durationAdjusted, map[string]any{
		"timeLeft":      t.timeLeft,
		"breakDuration": t.breakDuration,
		"focusDuration": t.focusDuration,
	})
}

func (t *timer) adjustSessionDuration(delta int) {
	//Updated duration should not be less than 300s (5min)
	calcDuration := func(duration, delta int) int {
		if duration+delta < 300 {
			return 300
		}
		return duration + delta
	}

	if t.sessionType == sessionBreak {
		t.breakDuration = calcDuration(t.breakDuration, delta)
	} else {
		t.focusDuration = calcDuration(t.focusDuration, delta)
	}

	t.sendUpdate(durationAdjusted, map[string]any{
		"timeLeft":      t.timeLeft,
		"breakDuration": t.breakDuration,
		"focusDuration": t.focusDuration,
	})
}

func (t *timer) readState() timerState {
	t.mu.Lock()
	stateCopy := t.stateCache
	t.mu.Unlock()

	return stateCopy
}

func (t *timer) sendUpdate(name string, args map[string]any) {
	slog.Debug(fmt.Sprintf("timer: update pending - name: %s, args: %v", name, args))

	select {
	case <-t.ctx.Done():
		return
	case t.updates <- Update{Name: name, Args: args}:
		slog.Debug(fmt.Sprintf("timer: update sent - name: %s, args: %v", name, args))
	}
}

func (t *timer) close() {
	t.ticker.Stop()
}
