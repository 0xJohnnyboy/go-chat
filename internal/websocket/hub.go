package websocket

import (
	"sync"

	"go-chat/pkg/chat"
)

type Hub struct {
	clients    map[*Client]bool
	channels   map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan HubMessage
	mu         sync.RWMutex
}

type HubMessage struct {
	ChannelID     string
	Message       chat.WebSocketMessage
	ExcludeClient *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		channels:   make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan HubMessage),
	}
}

func (h *Hub) Run() {
	
}

func (h *Hub) RegisterClient(client *Client) {
	
}

func (h *Hub) UnregisterClient(client *Client) {
	
}

func (h *Hub) JoinChannel(client *Client, channelID string) {
	
}

func (h *Hub) LeaveChannel(client *Client, channelID string) {
	
}

func (h *Hub) BroadcastToChannel(channelID string, message chat.WebSocketMessage, excludeClient *Client) {
	
}

func (h *Hub) BroadcastToUser(userID string, message chat.WebSocketMessage) {
	
}

func (h *Hub) GetChannelClients(channelID string) []*Client {
	return nil
}

func (h *Hub) GetClientCount() int {
	return 0
}

func (h *Hub) GetChannelClientCount(channelID string) int {
	return 0
}

func (h *Hub) IsClientInChannel(client *Client, channelID string) bool {
	return false
}

func (h *Hub) GetClientByUserID(userID string) *Client {
	return nil
}