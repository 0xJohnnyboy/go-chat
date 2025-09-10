package websocket

import (
	"log"
	"net/http"
	"sync"
	"time"

	"go-chat/pkg/chat"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking for production
		return true
	},
}

// Client represents a WebSocket client connection
type Client struct {
	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// The hub this client is connected to.
	hub *Hub

	// User information
	userID   string
	username string

	// Channels this client has joined
	channels map[string]bool
	mu       sync.RWMutex

	// Connection metadata
	connectedAt time.Time
	lastSeen    time.Time
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID, username string) *Client {
	return &Client{
		conn:        conn,
		send:        make(chan []byte, 256),
		hub:         hub,
		userID:      userID,
		username:    username,
		channels:    make(map[string]bool),
		connectedAt: time.Now(),
		lastSeen:    time.Now(),
	}
}

// GetUserID returns the user ID
func (c *Client) GetUserID() string {
	return c.userID
}

// GetUsername returns the username
func (c *Client) GetUsername() string {
	return c.username
}

// GetChannels returns a copy of joined channels
func (c *Client) GetChannels() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	channels := make([]string, 0, len(c.channels))
	for channelID := range c.channels {
		channels = append(channels, channelID)
	}
	return channels
}

// JoinChannel adds the client to a channel
func (c *Client) JoinChannel(channelID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channels[channelID] = true
}

// LeaveChannel removes the client from a channel
func (c *Client) LeaveChannel(channelID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.channels, channelID)
}

// IsInChannel checks if client is in a specific channel
func (c *Client) IsInChannel(channelID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.channels[channelID]
}

// UpdateLastSeen updates the last seen timestamp
func (c *Client) UpdateLastSeen() {
	c.lastSeen = time.Now()
}

// SendMessage sends a WebSocket message to the client
func (c *Client) SendMessage(message chat.WebSocketMessage) error {
	// TODO: Implement message serialization and sending
	return nil
}

// ReadPump pumps messages from the websocket connection to the hub.
func (c *Client) ReadPump() {
	// TODO: Implement message reading from WebSocket
}

// WritePump pumps messages from the hub to the websocket connection.
func (c *Client) WritePump() {
	// TODO: Implement message writing to WebSocket
}

// Close closes the client connection
func (c *Client) Close() {
	// TODO: Implement connection cleanup
}