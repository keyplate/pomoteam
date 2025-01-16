package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTimerStarted(t *testing.T) {
    messages := make(chan timerUpdate, 1)
    done := make(chan bool, 1)
    timer := &timer{
        id: uuid.New(),
        alias: nil,
        ticker: time.NewTicker(1 * time.Millisecond),
        messages: messages,
        done: done,
    }
    go timer.start(5)
    msg := <- timer.messages
    if msg.name != start {
        t.Fatalf("Received invalid start message, msg: %v", msg.name)
    }
}

func TestTimerTimeOuted(t *testing.T) {
    messages := make(chan timerUpdate, 1)
    done := make(chan bool, 1)
    timer := &timer{
        id: uuid.New(),
        alias: nil,
        ticker: time.NewTicker(1 * time.Millisecond),
        messages: messages,
        done: done,
    }
    go timer.start(5)
    for i := 0; i < 7; i++ {
        msg := <- timer.messages
        if i != 6 {
            continue
        }
        if msg.name != timeOut {
            t.Fatalf("Last message was not timeout %s", msg.name)
        }
    }
}
