package server

import (
	"sync"
	"go-chat/pkg/chat"
)

type Hub struct {
	mu sync.Mutex

	Clients map[string]*Client
	Channels map[string]*Channel
	Joined map[string]string
}

func NewHub() *Hub {
	return &Hub{
		Clients:  make(map[string]*Client),
		Channels: make(map[string]*chat.Channel),
		Joined:   make(map[string]string),
	}
}
