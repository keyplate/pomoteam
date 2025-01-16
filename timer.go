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

type timer struct {
    id uuid.UUID
    alias *string
    ticker *time.Ticker
    messages chan timerUpdate
    duration int
    currentTime int
    done chan bool
}

type timerUpdate struct {
    timerId uuid.UUID
    name string
    message string
}

func new(id uuid.UUID, alias *string, duration int) *timer {
    ticker := time.NewTicker(1 * time.Second)
    messages := make(chan timerUpdate)
    done := make(chan bool)
    return &timer{
        id: id,
        alias: alias,
        ticker: ticker,
        duration: duration,
        currentTime: 0,
        messages: messages,
        done: done,
    }
}

func (t *timer)start(duration int) {
    t.duration = duration
    t.currentTime = 0
    
    countdown := func() {
        for t.currentTime < t.duration {
        select {
            case <- t.ticker.C:
                t.currentTime++
                t.messages <- timerUpdate{ timerId: t.id, name: update, message: strconv.Itoa(t.currentTime) }
            case <- t.done:
                return
            }
        }
        t.messages <- timerUpdate{ timerId: t.id, name: timeOut }
    }
    t.messages <- timerUpdate{ timerId: t.id, name: start }
    go countdown()
}

func (t *timer)pause() {
    t.ticker.Stop()
    t.messages <- timerUpdate{ timerId: t.id, name: pause }
}

func (t *timer)resume() {
    t.ticker.Reset(1 * time.Second)
    t.messages <- timerUpdate{ timerId: t.id, name: resume }
}
