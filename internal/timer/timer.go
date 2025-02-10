//This package represents a concurrent timer for the flow-time techenque (which is a bit more agile version of pomodoro).
//Timer is operated by using commands. Timer publishes updates notifying users about its state changes.
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
	//commands
	start  = "START"
	stop   = "STOP"
	resume = "RESUME"
	adjust = "ADJUST"
)

const (
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
	isActive      bool
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
		isActive:      false,
		sessionType:   sessionType,
	}

	go timer.listenCommands()
	return timer
}

//start runs the timer.
//First duration evaluated based on the current sessionType, users
//are notified about the start when started, and timeout when timer ran out of time.
//After each timer circle session is switched to the opposite.
func (t *timer) start() {
	if t.isActive {
		return
	}

	t.timeLeft = int64(t.focusDuration)
	if t.sessionType == sessionBreak {
		t.timeLeft = int64(t.breakDuration)
	}

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
				t.tick()
			case <-t.done:
				return
			}
		}

		t.switchSession()
		t.updates <- timerUpdate{Name: timeOut}
	}
	go countdown()
}

//tick represents a logical tick of the timer and an actual passage of 1s. 
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
	if t.sessionType == sessionBreak {
		t.sessionType = sessionFocus
		return
	}
	t.sessionType = sessionBreak
}

func (t *timer) stop() {
	t.isActive = false
	t.ticker.Stop()

	t.updates <- timerUpdate{Name: stopped}
}

func (t *timer) resume() {
	t.isActive = true
	t.ticker.Reset(1 * time.Second)

	t.updates <- timerUpdate{Name: resumed}
}

//adjust alters timer duratin by given delta.
//If timer is running at the moment of adjustment, only currentTime is affected
//as it is assumed that user wants to alter only current session duration.
//If timer is not running the current session duration is affected.
func (t *timer) adjust(delta int) {
    if t.isActive {
	    t.timeLeft += int64(delta) 
        t.updates <- timerUpdate{Name: durationAdjusted}
        return
    } 

    if t.sessionType == sessionBreak {
        t.breakDuration += delta
    } else {
        t.focusDuration += delta
    }

	t.updates <- timerUpdate{
		Name: durationAdjusted,
		Args: map[string]string{
			"timeLeft": strconv.Itoa(int(t.timeLeft)),
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
	case stop:
		t.stop()
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
