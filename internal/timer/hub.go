package timer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	// Update event types
	timeUpdate       = "TIME_UPDATE"
	timeOut          = "TIME_OUT"
	started          = "STARTED"
	stopped          = "STOPPED"
	durationAdjusted = "DURATION_ADJUSTED"
	closed           = "CLOSED"
	resumed          = "RESUMED"
	paused           = "PAUSED"
	sessionUpdate    = "SESSION_UPDATE"
	timerReset       = "TIMER_RESET"
)

const (
	// Command types
	start  = "START"
	stop   = "STOP"
	pause  = "PAUSE"
	resume = "RESUME"
	adjust = "ADJUST"
	reset  = "RESET"
)

const (
	connect    = "CONNECT"
	disconnect = "DISCONNECT"
	state      = "STATE"
)

type Update struct {
	Name string            `json:"name"`
	Args map[string]string `json:"args,omitempty"`
}

type Command struct {
	Name string            `jsong:"name"`
	Args map[string]string `json:"args,omitempty"`
}

type hub struct {
	id            uuid.UUID
	timer         *timer
	clients       map[*Client]bool
	commands      chan Command
	updates       chan Update
	register      chan *Client
	unregister    chan *Client
	unregisterHub func(uuid.UUID)
	ctx           context.Context
	cancel        context.CancelFunc
}

func newHub(unregisterHub func(uuid.UUID)) *hub {
	ctx, cancel := context.WithCancel(context.Background())
	updates := make(chan Update)

	return &hub{
		id:            uuid.New(),
		timer:         NewTimer(ctx, updates),
		clients:       map[*Client]bool{},
		commands:      make(chan Command),
		updates:       updates,
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		ctx:           ctx,
		cancel:        cancel,
		unregisterHub: unregisterHub,
	}
}

func (h *hub) run() {
	go h.listenCommands()

	for {
		select {
		case upd := <-h.updates:
			slog.Debug(fmt.Sprintf("hub: id %v - received update", h.id))
			h.broadcast(upd)

		case client := <-h.register:
			slog.Debug(fmt.Sprintf("hub: id %v - received registeration", h.id))
			h.registerClient(client)

		case client := <-h.unregister:
			slog.Debug(fmt.Sprintf("hub: id %v - received unregistration", h.id))
			h.unregisterClient(client)

		case <-h.ctx.Done():
			h.closeHub()
			return
		}
	}
}

func (h *hub) listenCommands() {
	for {
		select {
		case command := <-h.commands:
			err := h.handleCommand(command)
			if err != nil {
				slog.Info(fmt.Sprintf("hub: id %v - error during command processing %v", h.id, err))
			}
		case <-h.ctx.Done():
			return
		}
	}
}

func (h *hub) handleCommand(cmd Command) error {
	switch cmd.Name {
	case start:
		h.timer.start()
	case pause:
		h.timer.pause()
	case resume:
		h.timer.resume()
	case reset:
		h.timer.reset()
	case adjust:
		duration, err := strconv.Atoi(cmd.Args["duration"])
		if err != nil {
			return errors.New("timer parseCommand: can not parse timer duration")
		}
		h.timer.adjust(duration)
	default:
		return fmt.Errorf("unknown command: %v", cmd)
	}
	return nil
}

func (h *hub) registerClient(client *Client) {
	client.send <- h.state()
	update := Update{
		Name: connect,
	}
	h.broadcast(update)
	h.clients[client] = true

	slog.Info(fmt.Sprintf("hub: id %v - registered client", h.id))
}

func (h *hub) unregisterClient(client *Client) {
	update := Update{
		Name: disconnect,
	}
	h.broadcast(update)
	delete(h.clients, client)

	slog.Info(fmt.Sprintf("hub: id %v - unregistered client", h.id))

	if len(h.clients) == 0 {
		go h.scheduleClose()
	}
}

func (h *hub) closeHub() {
	close(h.register)
	h.timer.close()
	close(h.commands)
	close(h.unregister)
	h.unregisterHub(h.id)

	slog.Info(fmt.Sprintf("hub: id %v - closed", h.id))
}

func (h *hub) broadcast(update Update) {
	for client := range h.clients {
		client.send <- update
	}
}

func (h *hub) state() Update {
	return Update{
		Name: state,
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
	h.cancel()

	slog.Info(fmt.Sprintf("hub: id %v - closing", h.id))
}
