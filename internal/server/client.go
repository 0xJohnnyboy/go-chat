package server

import (
	"github.com/gorilla/websocket"
	"go-chat/pkg/chat"
)

type Client struct {
	Conn      *websocket.Conn
	User      *chat.User
	ChannelID string
	Outgoing  chan []byte
}

func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Outgoing:
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.Conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("WebSocket writer error: %v", err)
				return
			}
		}
	}
}
