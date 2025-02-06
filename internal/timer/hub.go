package timer

import "github.com/google/uuid"

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
		case update := <-h.timer.updates:
			h.broadcast(update)
		case command := <-h.commands:
			h.timer.commands <- command
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			delete(h.clients, client)
		case <-h.done:
			h.timer.Close()
			close(h.commands)
			close(h.register)
			close(h.unregister)
		}
	}
}

func (h *hub) broadcast(update timerUpdate) {
	for client := range h.clients {
		client.send <- update
	}
}
