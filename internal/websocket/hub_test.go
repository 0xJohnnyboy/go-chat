package websocket

import (
	"sync"
	"testing"
	"time"

	"go-chat/pkg/chat"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	
	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.channels)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
}

func TestHub_RegisterClient(t *testing.T) {
	hub := NewHub()
	mockConn := &websocket.Conn{}
	
	client := NewClient(hub, mockConn, "user123", "testuser")
	
	hub.RegisterClient(client)
	
	assert.Equal(t, 1, hub.GetClientCount())
}

func TestHub_UnregisterClient(t *testing.T) {
	hub := NewHub()
	mockConn := &websocket.Conn{}
	
	client := NewClient(hub, mockConn, "user123", "testuser")
	
	hub.RegisterClient(client)
	assert.Equal(t, 1, hub.GetClientCount())
	
	hub.UnregisterClient(client)
	assert.Equal(t, 0, hub.GetClientCount())
}

func TestHub_JoinChannel(t *testing.T) {
	hub := NewHub()
	mockConn := &websocket.Conn{}
	
	client := NewClient(hub, mockConn, "user123", "testuser")
	hub.RegisterClient(client)
	
	hub.JoinChannel(client, "channel1")
	
	assert.True(t, hub.IsClientInChannel(client, "channel1"))
	assert.Equal(t, 1, hub.GetChannelClientCount("channel1"))
}

func TestHub_LeaveChannel(t *testing.T) {
	hub := NewHub()
	mockConn := &websocket.Conn{}
	
	client := NewClient(hub, mockConn, "user123", "testuser")
	hub.RegisterClient(client)
	hub.JoinChannel(client, "channel1")
	
	assert.True(t, hub.IsClientInChannel(client, "channel1"))
	
	hub.LeaveChannel(client, "channel1")
	
	assert.False(t, hub.IsClientInChannel(client, "channel1"))
	assert.Equal(t, 0, hub.GetChannelClientCount("channel1"))
}

func TestHub_BroadcastToChannel(t *testing.T) {
	hub := NewHub()
	mockConn1 := &websocket.Conn{}
	mockConn2 := &websocket.Conn{}
	mockConn3 := &websocket.Conn{}
	
	client1 := NewClient(hub, mockConn1, "user1", "user1")
	client2 := NewClient(hub, mockConn2, "user2", "user2")
	client3 := NewClient(hub, mockConn3, "user3", "user3")
	
	hub.RegisterClient(client1)
	hub.RegisterClient(client2)
	hub.RegisterClient(client3)
	
	hub.JoinChannel(client1, "channel1")
	hub.JoinChannel(client2, "channel1")
	hub.JoinChannel(client3, "channel2")
	
	message := chat.WebSocketMessage{
		Type: chat.MessageTypeMessage,
		Data: chat.MessagePayload{
			MessageID: "msg123",
			ChannelID: "channel1",
			Content:   "Hello, world!",
			UserID:    "user1",
			Username:  "user1",
			CreatedAt: time.Now(),
		},
		Timestamp: time.Now(),
	}
	
	hub.BroadcastToChannel("channel1", message, nil)
	
	assert.Equal(t, 2, hub.GetChannelClientCount("channel1"))
}

func TestHub_BroadcastToChannelWithExclusion(t *testing.T) {
	hub := NewHub()
	mockConn1 := &websocket.Conn{}
	mockConn2 := &websocket.Conn{}
	
	client1 := NewClient(hub, mockConn1, "user1", "user1")
	client2 := NewClient(hub, mockConn2, "user2", "user2")
	
	hub.RegisterClient(client1)
	hub.RegisterClient(client2)
	
	hub.JoinChannel(client1, "channel1")
	hub.JoinChannel(client2, "channel1")
	
	message := chat.WebSocketMessage{
		Type: chat.MessageTypeMessage,
		Data: chat.MessagePayload{
			MessageID: "msg123",
			ChannelID: "channel1",
			Content:   "Hello, world!",
			UserID:    "user1",
			Username:  "user1",
			CreatedAt: time.Now(),
		},
		Timestamp: time.Now(),
	}
	
	hub.BroadcastToChannel("channel1", message, client1)
	
	assert.Equal(t, 2, hub.GetChannelClientCount("channel1"))
}

func TestHub_BroadcastToUser(t *testing.T) {
	hub := NewHub()
	mockConn := &websocket.Conn{}
	
	client := NewClient(hub, mockConn, "user123", "testuser")
	hub.RegisterClient(client)
	
	message := chat.WebSocketMessage{
		Type: chat.MessageTypeError,
		Data: chat.ErrorPayload{
			Code:    "PERMISSION_DENIED",
			Message: "You don't have permission to perform this action",
		},
		Timestamp: time.Now(),
	}
	
	hub.BroadcastToUser("user123", message)
}

func TestHub_GetChannelClients(t *testing.T) {
	hub := NewHub()
	mockConn1 := &websocket.Conn{}
	mockConn2 := &websocket.Conn{}
	mockConn3 := &websocket.Conn{}
	
	client1 := NewClient(hub, mockConn1, "user1", "user1")
	client2 := NewClient(hub, mockConn2, "user2", "user2")
	client3 := NewClient(hub, mockConn3, "user3", "user3")
	
	hub.RegisterClient(client1)
	hub.RegisterClient(client2)
	hub.RegisterClient(client3)
	
	hub.JoinChannel(client1, "channel1")
	hub.JoinChannel(client2, "channel1")
	hub.JoinChannel(client3, "channel2")
	
	clients := hub.GetChannelClients("channel1")
	assert.Len(t, clients, 2)
	
	userIDs := make([]string, len(clients))
	for i, client := range clients {
		userIDs[i] = client.GetUserID()
	}
	assert.Contains(t, userIDs, "user1")
	assert.Contains(t, userIDs, "user2")
	assert.NotContains(t, userIDs, "user3")
}

func TestHub_GetClientByUserID(t *testing.T) {
	hub := NewHub()
	mockConn := &websocket.Conn{}
	
	client := NewClient(hub, mockConn, "user123", "testuser")
	hub.RegisterClient(client)
	
	foundClient := hub.GetClientByUserID("user123")
	assert.NotNil(t, foundClient)
	assert.Equal(t, "user123", foundClient.GetUserID())
	assert.Equal(t, "testuser", foundClient.GetUsername())
	
	notFoundClient := hub.GetClientByUserID("nonexistent")
	assert.Nil(t, notFoundClient)
}

func TestHub_MultipleChannels(t *testing.T) {
	hub := NewHub()
	mockConn1 := &websocket.Conn{}
	mockConn2 := &websocket.Conn{}
	
	client1 := NewClient(hub, mockConn1, "user1", "user1")
	client2 := NewClient(hub, mockConn2, "user2", "user2")
	
	hub.RegisterClient(client1)
	hub.RegisterClient(client2)
	
	hub.JoinChannel(client1, "channel1")
	hub.JoinChannel(client1, "channel2")
	hub.JoinChannel(client2, "channel1")
	
	assert.True(t, hub.IsClientInChannel(client1, "channel1"))
	assert.True(t, hub.IsClientInChannel(client1, "channel2"))
	assert.True(t, hub.IsClientInChannel(client2, "channel1"))
	assert.False(t, hub.IsClientInChannel(client2, "channel2"))
	
	assert.Equal(t, 2, hub.GetChannelClientCount("channel1"))
	assert.Equal(t, 1, hub.GetChannelClientCount("channel2"))
}

func TestHub_ConcurrentAccess(t *testing.T) {
	hub := NewHub()
	
	const numClients = 100
	const numChannels = 10
	
	var wg sync.WaitGroup
	
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			
			mockConn := &websocket.Conn{}
			client := NewClient(hub, mockConn, "user"+string(rune('0'+clientID)), "user"+string(rune('0'+clientID)))
			
			hub.RegisterClient(client)
			
			for j := 0; j < numChannels; j++ {
				channelID := "channel" + string(rune('0'+(clientID%3)))
				hub.JoinChannel(client, channelID)
			}
		}(i)
	}
	
	wg.Wait()
	
	assert.Equal(t, numClients, hub.GetClientCount())
	
	totalChannelClients := 0
	for i := 0; i < 3; i++ {
		channelID := "channel" + string(rune('0'+i))
		count := hub.GetChannelClientCount(channelID)
		totalChannelClients += count
	}
	
	assert.True(t, totalChannelClients > 0)
}

func TestHub_UnregisterClientFromAllChannels(t *testing.T) {
	hub := NewHub()
	mockConn := &websocket.Conn{}
	
	client := NewClient(hub, mockConn, "user123", "testuser")
	hub.RegisterClient(client)
	
	hub.JoinChannel(client, "channel1")
	hub.JoinChannel(client, "channel2")
	hub.JoinChannel(client, "channel3")
	
	assert.Equal(t, 1, hub.GetChannelClientCount("channel1"))
	assert.Equal(t, 1, hub.GetChannelClientCount("channel2"))
	assert.Equal(t, 1, hub.GetChannelClientCount("channel3"))
	
	hub.UnregisterClient(client)
	
	assert.Equal(t, 0, hub.GetClientCount())
	assert.Equal(t, 0, hub.GetChannelClientCount("channel1"))
	assert.Equal(t, 0, hub.GetChannelClientCount("channel2"))
	assert.Equal(t, 0, hub.GetChannelClientCount("channel3"))
}

func TestHub_EmptyChannelHandling(t *testing.T) {
	hub := NewHub()
	
	assert.Equal(t, 0, hub.GetChannelClientCount("nonexistent"))
	assert.Empty(t, hub.GetChannelClients("nonexistent"))
	assert.False(t, hub.IsClientInChannel(nil, "nonexistent"))
}

func TestHub_NilClientHandling(t *testing.T) {
	hub := NewHub()
	
	hub.RegisterClient(nil)
	assert.Equal(t, 0, hub.GetClientCount())
	
	hub.UnregisterClient(nil)
	assert.Equal(t, 0, hub.GetClientCount())
	
	hub.JoinChannel(nil, "channel1")
	assert.Equal(t, 0, hub.GetChannelClientCount("channel1"))
	
	hub.LeaveChannel(nil, "channel1")
	assert.Equal(t, 0, hub.GetChannelClientCount("channel1"))
}