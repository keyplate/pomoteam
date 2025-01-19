package main

import (
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
    update = "UPDATE"
    timeOut = "TIME_OUT"
    start = "START"
    stop = "STOP"
    pause = "PAUSE"
    resume = "RESUME"
)

type Timer struct {
    id uuid.UUID
    ticker *time.Ticker
    messages chan TimerUpdate
    duration int
    currentTime int
    done chan bool
}

type TimerUpdate struct {
    timerId uuid.UUID
    name string
    message string
}

func New(id uuid.UUID) *Timer {
    ticker := time.NewTicker(1 * time.Second)
    messages := make(chan TimerUpdate, 1)
    done := make(chan bool, 1)
    return &Timer{
        id: id,
        ticker: ticker,
        duration: 0,
        currentTime: 0,
        messages: messages,
        done: done,
    }
}

func (t *Timer)Start(duration int) {
    t.duration = duration
    t.currentTime = 0
    
    countdown := func() {
        for t.currentTime < t.duration {
        select {
            case <- t.ticker.C:
                t.currentTime++
                t.messages <- TimerUpdate{ timerId: t.id, name: update, message: strconv.Itoa(t.currentTime) }
            case <- t.done:
                return
            }
        }
        t.messages <- TimerUpdate{ timerId: t.id, name: timeOut }
    }
    t.messages <- TimerUpdate{ timerId: t.id, name: start }
    go countdown()
}

func (t *Timer)Pause() {
    t.ticker.Stop()
    t.messages <- TimerUpdate{ timerId: t.id, name: pause }
}

func (t *Timer)Resume() {
    t.ticker.Reset(1 * time.Second)
    t.messages <- TimerUpdate{ timerId: t.id, name: resume }
}

func (t *Timer)Subscribe() (chan TimerUpdate, chan bool) {
    return t.messages, t.done
}
