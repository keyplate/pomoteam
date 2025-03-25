package timer

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	connect    = "CONNECT"
	disconnect = "DISCONNECT"
	state      = "STATE"
)

type update struct {
	UpdateType string            `json:"type"`
	Name       string            `json:"name"`
	Args       map[string]string `json:"args"`
}

type hub struct {
	id            uuid.UUID
	timer         *timer
	clients       map[*Client]bool
	commands      chan timerCommand
	register      chan *Client
	unregister    chan *Client
	unregisterHub chan uuid.UUID
	done          chan struct{}
}

func newHub(timer *timer, unregisterHub chan uuid.UUID) *hub {
	return &hub{
		id:            uuid.New(),
		timer:         timer,
		clients:       map[*Client]bool{},
		commands:      make(chan timerCommand),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		done:          make(chan struct{}),
		unregisterHub: unregisterHub,
	}
}

func (h *hub) run() {
	for {
		select {
		case timerUpdate := <-h.timer.updates:
			slog.Debug(fmt.Sprintf("hub: id %v - received update", h.id))
			h.sendUpdate(timerUpdate)

		case command := <-h.commands:
			slog.Debug(fmt.Sprintf("hub: id %v - received command", h.id))
			h.timer.commands <- command

		case client := <-h.register:
			slog.Debug(fmt.Sprintf("hub: id %v - received registeration", h.id))
			h.registerClient(client)

		case client := <-h.unregister:
			slog.Debug(fmt.Sprintf("hub: id %v - received unregistration", h.id))
			h.unregisterClient(client)

		case <-h.done:
			h.closeHub()
			return
		}
	}
}

func (h *hub) sendUpdate(timerUpdate timerUpdate) {
	slog.Debug(fmt.Sprintf("hub: id %v - sending update", h.id))

	update := update{
		UpdateType: "timer",
		Name:       timerUpdate.Name,
		Args:       timerUpdate.Args,
	}
	h.broadcast(update)

	slog.Debug(fmt.Sprintf("hub: id %v - update is sent", h.id))
}

func (h *hub) registerClient(client *Client) {
	client.send <- h.state()
	update := update{
		UpdateType: "hub",
		Name:       connect,
	}
	h.broadcast(update)
	h.clients[client] = true

	slog.Info(fmt.Sprintf("hub: id %v - registered client", h.id))
}

func (h *hub) unregisterClient(client *Client) {
	update := update{
		UpdateType: "hub",
		Name:       disconnect,
	}
	h.broadcast(update)
	delete(h.clients, client)

	slog.Info(fmt.Sprintf("hub: id %v - unregistered client", h.id))

	if len(h.clients) == 0 {
		go h.scheduleClose()
	}
}

func (h *hub) closeHub() {
	h.timer.Close()
	close(h.commands)
	close(h.register)
	close(h.unregister)
	h.unregisterHub <- h.id
	close(h.done)

	slog.Info(fmt.Sprintf("hub: id %v - closed", h.id))
}

func (h *hub) broadcast(update update) {
	for client := range h.clients {
		client.send <- update
	}
}

func (h *hub) state() update {
	return update{
		UpdateType: "hub",
		Name:       state,
		Args: map[string]string{
			"focusDuration":  strconv.Itoa(h.timer.focusDuration),
			"breakDuration":  strconv.Itoa(h.timer.breakDuration),
			"sessionType":    h.timer.sessionType,
			"isRunning":      strconv.FormatBool(h.timer.isRunning),
			"isSessionEnded": strconv.FormatBool(h.timer.isSessionEnded),
		},
	}
}

func (h *hub) scheduleClose() {
	slog.Info(fmt.Sprintf("hub: id %v - closure scheduled", h.id))

	timer := time.NewTimer(120 * time.Second)
	<-timer.C

	if len(h.clients) != 0 {
		return
	}
	h.done <- struct{}{}

	slog.Info(fmt.Sprintf("hub: id %v - closing", h.id))
}
