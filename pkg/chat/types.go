package chat

import (
	"sync"

	"github.com/gorilla/websocket"
)

type WSMessage struct {
	ChannelID string `json:"channel_id"`
	SenderID  string `json:"sender_id"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

type Client struct {
	Conn *websocket.Conn
	User *User
	Outgoing chan []WSMessage
}

type Hub struct {
	mu sync.Mutex

	Clients map[string]*Client
	Channels map[string]*Channel
	Joined map[string]string
}

func NewHub() *Hub {
	return &Hub{
		Clients:  make(map[string]*Client),
		Channels: make(map[string]*Channel),
		Joined:   make(map[string]string),
	}
}
