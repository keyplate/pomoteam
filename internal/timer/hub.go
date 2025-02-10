package timer

import (
	"strconv"

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
	id         uuid.UUID
	timer      *timer
	clients    map[*Client]bool
	commands   chan timerCommand
	register   chan *Client
	unregister chan *Client
	done       chan bool
}

func newHub(timer *timer) *hub {
	return &hub{
		id:         uuid.New(),
		timer:      timer,
		clients:    map[*Client]bool{},
		commands:   make(chan timerCommand),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		done:       make(chan bool),
	}
}

func (h *hub) run() {
	for {
		select {
		case timerUpdate := <-h.timer.updates:
            h.sendUpdate(timerUpdate)

		case command := <-h.commands:
			h.timer.commands <- command

		case client := <-h.register:
            h.registerClient(client)

		case client := <-h.unregister:
            h.unregisterClient(client)

		case <-h.done:
            h.closeHub()
		}
	}
}

func (h *hub) sendUpdate(timerUpdate timerUpdate) {
	update := update{
		UpdateType: "timer",
		Name:       timerUpdate.Name,
		Args:       timerUpdate.Args,
	}
	h.broadcast(update)
}

func (h *hub) registerClient(client *Client) {
    client.send <- h.state() 
    update := update{
        UpdateType: "hub",
        Name:       connect,
    }
    h.broadcast(update)
    h.clients[client] = true
}

func (h *hub) unregisterClient(client *Client) {
    update := update{
        UpdateType: "hub",
        Name:       disconnect,
    }
    h.broadcast(update)
    delete(h.clients, client)
}

func (h *hub) closeHub() {
    h.timer.Close()
    close(h.commands)
    close(h.register)
    close(h.unregister)
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
			"focusDuration": strconv.Itoa(h.timer.focusDuration),
			"breakDuration": strconv.Itoa(h.timer.breakDuration),
			"sessionType":   h.timer.sessionType,
			"isRunning":     strconv.FormatBool(h.timer.isActive),
		},
	}
}
