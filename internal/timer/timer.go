// This package represents a concurrent timer for the flow-time techenque (which is a bit more agile version of pomodoro).
// Timer is operated by using commands. Timer publishes updates notifying users about its state changes.
package timer

import (
	"errors"
	"log"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	//updates
	timeUpdate       = "TIME_UPDATE"
	timeOut          = "TIME_OUT"
	started          = "STARTED"
	stopped          = "STOPPED"
	durationAdjusted = "DURATION_ADJUSTED"
	closed           = "CLOSED"
	resumed          = "RESUMED"
    paused          = "PAUSED"
	sessionUpdate    = "SESSION_UPDATE"
	//commands
	start  = "START"
	stop   = "STOP"
    pause  = "PAUSE"
	resume = "RESUME"
	adjust = "ADJUST"
	//session types
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

type timer struct {
	ticker   *time.Ticker
	updates  chan timerUpdate
	commands chan timerCommand
	done     chan bool

	breakDuration int
	focusDuration int
	timeLeft      int64
	isRunning     bool
	sessionType   string
}

func NewTimer(breakDuration, focusDuration int, sessionType string) *timer {
	timer := &timer{
		ticker:   time.NewTicker(1 * time.Second),
		done:     make(chan bool),
		commands: make(chan timerCommand),
		updates:  make(chan timerUpdate),

		breakDuration: breakDuration,
		focusDuration: focusDuration,
		isRunning:     false,
		sessionType:   sessionType,
	}

	go timer.listenCommands()
	return timer
}

func (t *timer) start() {
	if t.isRunning {
		return
	}

	//duration is evaluated based on the current sessionType
	t.timeLeft = int64(t.focusDuration)
	if t.sessionType == sessionBreak {
		t.timeLeft = int64(t.breakDuration)
	}

	countdown := func() {
		defer func() {
			t.ticker.Stop()
		}()

		t.isRunning = true
		t.ticker.Reset(1 * time.Second)
		t.updates <- timerUpdate{Name: started, Args: map[string]string{"isRunning": strconv.FormatBool(t.isRunning)}}

		for t.timeLeft > 0 {
			select {
			case <-t.ticker.C:
				t.tick()
			case <-t.done:
				return
			}
		}

		//every successful timer run switches sessionType to the opposite
		t.switchSession()
		t.isRunning = false
		t.updates <- timerUpdate{Name: timeOut, Args: map[string]string{"isRunning": strconv.FormatBool(t.isRunning)}}
	}
	go countdown()
}

// tick represents a logical tick of the timer and an actual passage of 1s.
func (t *timer) tick() {
	atomic.AddInt64(&t.timeLeft, -1)

	timeLeft := int(t.timeLeft)
	duration := t.focusDuration
	if t.sessionType == sessionBreak {
		duration = t.breakDuration
	}

	t.updates <- timerUpdate{
		Name: timeUpdate,
		Args: map[string]string{
			"timeLeft": strconv.Itoa(timeLeft),
			"duration": strconv.Itoa(duration),
		},
	}
}

func (t *timer) switchSession() {
	defer func() {
		t.updates <- timerUpdate{
			Name: sessionUpdate,
			Args: map[string]string{"sessionType": t.sessionType},
		}
	}()
	if t.sessionType == sessionBreak {
		t.sessionType = sessionFocus
		return
	}
	t.sessionType = sessionBreak
}

func (t *timer) pause() {
	t.isRunning = false
	t.ticker.Stop()

	t.updates <- timerUpdate{Name: paused, Args: map[string]string{"isRunning": strconv.FormatBool(t.isRunning)}}
}

func (t *timer) resume() {
	t.isRunning = true
	t.ticker.Reset(1 * time.Second)

	t.updates <- timerUpdate{Name: resumed, Args: map[string]string{"isRunning": strconv.FormatBool(t.isRunning)}}
}

// adjust alters timer duratin by given delta.
func (t *timer) adjust(delta int) {
	//If timer is running at the moment of adjustment, only currentTime is affected
	//as it is assumed that user wants to alter only current session duration.
	if t.isRunning {
		atomic.AddInt64(&t.timeLeft, int64(delta))
		t.updates <- timerUpdate{Name: durationAdjusted}
		return
	}

	//If timer is not running the current session duration is affected.
	if t.sessionType == sessionBreak {
		t.breakDuration += delta
	} else {
		t.focusDuration += delta
	}

	t.updates <- timerUpdate{
		Name: durationAdjusted,
		Args: map[string]string{
			"timeLeft":      strconv.Itoa(int(t.timeLeft)),
			"breakDuration": strconv.Itoa(t.breakDuration),
			"focusDuration": strconv.Itoa(t.focusDuration),
		},
	}
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
