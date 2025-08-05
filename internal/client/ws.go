package client

import (
	"go-chat/pkg/chat"
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	tea "github.com/charmbracelet/bubbletea"
)

type WSClient struct {
	conn *websocket.Conn
	ch chan tea.Msg
}

func NewWSClient(ch chan tea.Msg) (*WSClient, error) {
	u := url.URL{Scheme: "ws", Host: "localhost:9876", Path: "/ws"}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

	if err != nil {
		return nil, err
	}

	return &WSClient{conn: conn, ch: ch}, nil
}

func (c *WSClient) Start() {
	go func() {
		for {
			_, data, err := c.conn.ReadMessage()
			if err != nil {
				log.Println("WS read error:", err)
				return
			}
			c.ch <- messageReceivedMsg(data)
		}
	}()
}


func (c *WSClient) Send(msg chat.Message) error {
	data, err := json.Marshal(msg)

	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *WSClient) Listen() tea.Cmd {
	return func() tea.Msg {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("WS read error:", err)
			time.Sleep(time.Second)
			return nil
		}

		return messageReceivedMsg(data)
	}
}
