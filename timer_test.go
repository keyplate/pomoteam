package main

import (
	"slices"
	"testing"
	"time"
)

func TestTimerStarted(t *testing.T) {
	messages := make(chan timerUpdate, 1)
	timer := &timer{
		ticker:   time.NewTicker(1 * time.Millisecond),
		messages: messages,
	}
	timer.start(5)
	msg := <-timer.messages
	if msg.name != currentTime {
		t.Fatalf("didn't recevie currentTime update: %v", msg.name)
	}
}

func TestTimerTimeOuted(t *testing.T) {
	messages := make(chan timerUpdate, 1)
	expectedMessages := []string{currentTime, currentTime, currentTime, timeOut}
	actualMessages := []string{}
	timer := &timer{
		ticker:   time.NewTicker(1 * time.Millisecond),
		messages: messages,
	}

	go timer.start(3)
	for i := 0; i < 4; i++ {
		msg := <-timer.messages
		actualMessages = append(actualMessages, msg.name)
	}

	isValid := slices.Equal(expectedMessages, actualMessages)
	if !isValid {
		t.Fatalf("didn't receive correct messages: %v", actualMessages)
	}
}

func TestTimerPaused(t *testing.T) {
	messages := make(chan timerUpdate, 1)
	done := make(chan bool)

	timer := &timer{
		ticker:   time.NewTicker(1 * time.Millisecond),
		messages: messages,
	}

	timer.start(5)
	timer.pause()
	go func() {
		time.Sleep(5 * time.Millisecond)
		done <- true
	}()

	select {
	case <-timer.messages:
		t.Fatalf("received message after pause")
	case <-timer.ticker.C:
		t.Fatalf("ticker works after pause")
	case <-done:
		return
	}
}

func TestTimerResumed(t *testing.T) {
	messages := make(chan timerUpdate, 1)
	done := make(chan bool)

	timer := &timer{
		ticker:   time.NewTicker(1 * time.Millisecond),
		messages: messages,
	}

	timer.start(1)
	timer.pause()
	timer.resume()

	go func() {
		time.Sleep(2 * time.Second)
		done <- true
	}()

	select {
	case <-timer.messages:
		return
	case <-done:
		t.Fatalf("timer didn't resume")
	}
}

func TestParserRoutineRunning(t *testing.T) {
	expectedMessage := currentTime
	done := make(chan bool, 1)
	timer := New()
	timer.commands <- timerCommand{name: start, arg: "3"}

	go func() {
		time.Sleep(2 * time.Second)
		done <- true
	}()

	select {
	case message := <-timer.messages:
		if message.name != expectedMessage {
			t.Fatal("wrong message")
		}
	case <-done:
		t.Fatalf("timeout, command was not parsed")
	}
}
