package timer

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
	CheckOrigin:     func(r *http.Request) bool { return true }, //todo: set up adequate cors
}

type Client struct {
	hub  *hub
	conn *websocket.Conn
	send chan update
}

func ServeWs(hs *HubService, w http.ResponseWriter, r *http.Request) {
	hubId, err := uuid.Parse(r.PathValue("hubId"))
	if err != nil {
		http.Error(w, "bad requset", 400)
		return
	}

	hub, err := hs.get(hubId)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan update)}
	hub.register <- client

	go client.read()
	go client.write()
}

func (c *Client) read() {
	defer func() {
		c.conn.Close()
		c.hub.unregister <- c
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		var command timerCommand
		err := c.conn.ReadJSON(&command)
		if err != nil {
			log.Printf("error: %v", err)
			break
		}
		c.hub.commands <- command
	}
}

func (c *Client) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.conn.Close()
		ticker.Stop()
		c.hub.unregister <- c
	}()

	for {
		select {
		case update := <-c.send:
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

func (c *Client) close() {
	close(c.send)
	err := c.conn.Close()
	if err != nil {
		log.Printf("error: %v", err)
	}
}
