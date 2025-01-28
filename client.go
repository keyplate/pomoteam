package main

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// time limit for writing to the peer
	writeWait = 2 * time.Second

	// time limit for reading next pong message to the peer
	pongWait = 60 * time.Second

	// ping peer with this period. Must be less thean pongWait
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	timer *timer
	conn  *websocket.Conn
}

func ServeWs(ts *TimerService, w http.ResponseWriter, r *http.Request) {
	timerId, err := uuid.Parse(r.PathValue("timerId"))
	if err != nil {
		http.Error(w, "bad requset", 400)
        log.Print(err)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	timer, err := ts.Get(timerId)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	client := &Client{timer: timer, conn: conn}
	go client.read()
	go client.write()
}

func (c *Client) read() {
	defer func() {
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		var command timerCommand
		err := c.conn.ReadJSON(&command)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		c.timer.commands <- command
	}
}

func (c *Client) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.conn.Close()
		ticker.Stop()
	}()

	for {
		select {
		case update := <-c.timer.updates:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := c.conn.WriteJSON(update)
			if err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
