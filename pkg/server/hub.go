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
