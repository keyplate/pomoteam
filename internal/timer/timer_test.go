package timer

import (
	"slices"
	"testing"
	"time"
)

func TestTimerStarted(t *testing.T) {
	updates := make(chan timerUpdate, 1)
	timer := &timer{
		ticker:  time.NewTicker(1 * time.Millisecond),
		updates: updates,
	}

	timer.start(5)
	upd := <-timer.updates
	if upd.Name != started {
		t.Fatalf("didn't recevie currentTime update: %v", upd.Name)
	}
}

func TestTimerTimeOuted(t *testing.T) {
	updates := make(chan timerUpdate, 1)
	expectedUpdates := []string{started, currentTime, currentTime, currentTime, timeOut}
	actualUpdates := []string{}
	timer := &timer{
		ticker:  time.NewTicker(1 * time.Millisecond),
		updates: updates,
	}

	go timer.start(3)
	for i := 0; i < 5; i++ {
		msg := <-timer.updates
		actualUpdates = append(actualUpdates, msg.Name)
	}

	isValid := slices.Equal(expectedUpdates, actualUpdates)
	if !isValid {
		t.Fatalf("didn't receive correct updates: %v", actualUpdates)
	}
}

func TestTimerStopped(t *testing.T) {
	updates := make(chan timerUpdate, 1)
	done := make(chan bool)

	timer := &timer{
		ticker:  time.NewTicker(1 * time.Millisecond),
		updates: updates,
	}

	go func() {
		timer.start(10)
		time.Sleep(5 * time.Millisecond)
		timer.stop()
		time.Sleep(10 * time.Millisecond)
		done <- true
	}()

	select {
	case upd := <-timer.updates:
		if upd.Name == timeOut {
			t.Fatalf("timer didn't pause")
		}
	case <-done:
		return
	}
}

func TestTimerResumed(t *testing.T) {
	timer := NewTimer()
	go func() {
		timer.start(2)
		time.Sleep(10 * time.Microsecond)
		timer.stop()
		timer.resume()
	}()

	go func() {
		time.Sleep(3 * time.Second)
		timer.done <- true
	}()

	select {
	case upd := <-timer.updates:
		if upd.Name == timeOut {
			return
		}
	case <-timer.done:
		t.Fatalf("timer didn't resume")
	}
}

func TestParserRoutineRunning(t *testing.T) {
	expectedMessage := started
	timer := NewTimer()
	timer.commands <- timerCommand{Name: start, Arg: "3"}

	go func() {
		time.Sleep(2 * time.Second)
		timer.done <- true
	}()

	select {
	case message := <-timer.updates:
		if message.Name != expectedMessage {
			t.Fatal("wrong message")
		}
	case <-timer.done:
		t.Fatalf("timeout, command was not parsed")
	}
}
