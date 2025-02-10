package timer

import "github.com/google/uuid"

const (
    connect    = "CONNECT"
    disconnect = "DISCONNECT"
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
            update := update{
                UpdateType: "timer",
                Name: timerUpdate.Name,
                Args: timerUpdate.Args,
            }
			h.broadcast(update)
		case command := <-h.commands:
			h.timer.commands <- command
		case client := <-h.register:
            update := update{
                UpdateType: "hub",
                Name: connect,
            }
            h.broadcast(update)
			h.clients[client] = true
		case client := <-h.unregister:
            update := update{
                UpdateType: "hub",
                Name: disconnect,
            }
            h.broadcast(update)
			delete(h.clients, client)
		case <-h.done:
			h.timer.Close()
			close(h.commands)
			close(h.register)
			close(h.unregister)
		}
	}
}

func (h *hub) broadcast(update update) {
	for client := range h.clients {
		client.send <- update
	}
}
