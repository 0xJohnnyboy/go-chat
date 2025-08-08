package server

import (
	"github.com/gorilla/websocket"
	. "go-chat/pkg/server"
	. "go-chat/pkg/chat"
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

func (c *Client) ReadPump(hub *Hub) {
	c.Conn.SetReadLimit(512)
	c.Conn.SetCloseHandler(func(code int, text string) error {
		hub.unregister <- c
		return nil
	})

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var incoming Message
		if err := json.Unmarshal(msg, &incoming); err != nil {
			log.Printf("erreur parse message: %v", err)
			continue
		}

		// On fixe l'expéditeur côté serveur
		incoming.SenderID = c.User.ID
		incoming.ChannelID = c.ChannelID

		// Envoi au hub
		hub.broadcast <- &incoming
	}
}

