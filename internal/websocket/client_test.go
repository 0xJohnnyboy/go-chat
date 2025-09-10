package websocket

import (
	"testing"
	"time"

	"go-chat/pkg/chat"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock WebSocket connection for testing
type MockConn struct {
	mock.Mock
}

func (m *MockConn) WriteMessage(messageType int, data []byte) error {
	args := m.Called(messageType, data)
	return args.Error(0)
}

func (m *MockConn) ReadMessage() (messageType int, p []byte, err error) {
	args := m.Called()
	return args.Int(0), args.Get(1).([]byte), args.Error(2)
}

func (m *MockConn) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConn) SetReadDeadline(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *MockConn) SetPongHandler(h func(string) error) {
	m.Called(h)
}

func (m *MockConn) SetReadLimit(limit int64) {
	m.Called(limit)
}

// Mock Hub for testing
type MockHub struct {
	mock.Mock
}

func (m *MockHub) RegisterClient(client *Client) {
	m.Called(client)
}

func (m *MockHub) UnregisterClient(client *Client) {
	m.Called(client)
}

func (m *MockHub) BroadcastToChannel(channelID string, message chat.WebSocketMessage, excludeClient *Client) {
	m.Called(channelID, message, excludeClient)
}

func TestNewClient(t *testing.T) {
	mockConn := &websocket.Conn{}
	mockHub := &MockHub{}
	
	client := NewClient(mockHub, mockConn, "user123", "testuser")
	
	assert.NotNil(t, client)
	assert.Equal(t, "user123", client.GetUserID())
	assert.Equal(t, "testuser", client.GetUsername())
	assert.NotNil(t, client.send)
	assert.NotNil(t, client.channels)
	assert.True(t, client.connectedAt.Before(time.Now().Add(time.Second)))
}

func TestClient_GetUserID(t *testing.T) {
	mockConn := &websocket.Conn{}
	mockHub := &MockHub{}
	
	client := NewClient(mockHub, mockConn, "user456", "testuser")
	
	assert.Equal(t, "user456", client.GetUserID())
}

func TestClient_GetUsername(t *testing.T) {
	mockConn := &websocket.Conn{}
	mockHub := &MockHub{}
	
	client := NewClient(mockHub, mockConn, "user123", "alice")
	
	assert.Equal(t, "alice", client.GetUsername())
}

func TestClient_JoinChannel(t *testing.T) {
	mockConn := &websocket.Conn{}
	mockHub := &MockHub{}
	
	client := NewClient(mockHub, mockConn, "user123", "testuser")
	
	client.JoinChannel("channel1")
	
	assert.True(t, client.IsInChannel("channel1"))
	channels := client.GetChannels()
	assert.Contains(t, channels, "channel1")
	assert.Len(t, channels, 1)
}

func TestClient_LeaveChannel(t *testing.T) {
	mockConn := &websocket.Conn{}
	mockHub := &MockHub{}
	
	client := NewClient(mockHub, mockConn, "user123", "testuser")
	
	client.JoinChannel("channel1")
	client.JoinChannel("channel2")
	
	assert.True(t, client.IsInChannel("channel1"))
	assert.True(t, client.IsInChannel("channel2"))
	
	client.LeaveChannel("channel1")
	
	assert.False(t, client.IsInChannel("channel1"))
	assert.True(t, client.IsInChannel("channel2"))
	
	channels := client.GetChannels()
	assert.NotContains(t, channels, "channel1")
	assert.Contains(t, channels, "channel2")
	assert.Len(t, channels, 1)
}

func TestClient_IsInChannel(t *testing.T) {
	mockConn := &websocket.Conn{}
	mockHub := &MockHub{}
	
	client := NewClient(mockHub, mockConn, "user123", "testuser")
	
	assert.False(t, client.IsInChannel("channel1"))
	
	client.JoinChannel("channel1")
	assert.True(t, client.IsInChannel("channel1"))
	
	client.LeaveChannel("channel1")
	assert.False(t, client.IsInChannel("channel1"))
}

func TestClient_GetChannels(t *testing.T) {
	mockConn := &websocket.Conn{}
	mockHub := &MockHub{}
	
	client := NewClient(mockHub, mockConn, "user123", "testuser")
	
	channels := client.GetChannels()
	assert.Empty(t, channels)
	
	client.JoinChannel("channel1")
	client.JoinChannel("channel2")
	client.JoinChannel("channel3")
	
	channels = client.GetChannels()
	assert.Len(t, channels, 3)
	assert.Contains(t, channels, "channel1")
	assert.Contains(t, channels, "channel2")
	assert.Contains(t, channels, "channel3")
}

func TestClient_UpdateLastSeen(t *testing.T) {
	mockConn := &websocket.Conn{}
	mockHub := &MockHub{}
	
	client := NewClient(mockHub, mockConn, "user123", "testuser")
	initialTime := client.lastSeen
	
	time.Sleep(10 * time.Millisecond)
	client.UpdateLastSeen()
	
	assert.True(t, client.lastSeen.After(initialTime))
}

func TestClient_SendMessage(t *testing.T) {
	mockConn := &websocket.Conn{}
	mockHub := &MockHub{}
	
	client := NewClient(mockHub, mockConn, "user123", "testuser")
	
	message := chat.WebSocketMessage{
		Type: chat.MessageTypeMessage,
		Data: chat.MessagePayload{
			MessageID: "msg123",
			ChannelID: "channel1",
			Content:   "Hello, world!",
			UserID:    "user456",
			Username:  "bob",
			CreatedAt: time.Now(),
		},
		Timestamp: time.Now(),
		MessageID: "msg123",
	}
	
	err := client.SendMessage(message)
	assert.NoError(t, err)
}

func TestClient_ConcurrentChannelAccess(t *testing.T) {
	mockConn := &websocket.Conn{}
	mockHub := &MockHub{}
	
	client := NewClient(mockHub, mockConn, "user123", "testuser")
	
	done := make(chan bool, 10)
	
	for i := 0; i < 5; i++ {
		go func(id int) {
			channelID := "channel" + string(rune('1'+id))
			client.JoinChannel(channelID)
			assert.True(t, client.IsInChannel(channelID))
			done <- true
		}(i)
	}
	
	for i := 0; i < 5; i++ {
		go func(id int) {
			channels := client.GetChannels()
			assert.True(t, len(channels) >= 0)
			done <- true
		}(i)
	}
	
	for i := 0; i < 10; i++ {
		<-done
	}
	
	channels := client.GetChannels()
	assert.Equal(t, 5, len(channels))
}

func TestClient_ChannelThreadSafety(t *testing.T) {
	mockConn := &websocket.Conn{}
	mockHub := &MockHub{}
	
	client := NewClient(mockHub, mockConn, "user123", "testuser")
	
	const numGoroutines = 100
	const numChannels = 10
	
	done := make(chan bool, numGoroutines*2)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numChannels; j++ {
				channelID := "channel" + string(rune('0'+j))
				client.JoinChannel(channelID)
			}
			done <- true
		}()
		
		go func() {
			for j := 0; j < numChannels; j++ {
				channelID := "channel" + string(rune('0'+j))
				client.IsInChannel(channelID)
			}
			done <- true
		}()
	}
	
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}
	
	channels := client.GetChannels()
	assert.Equal(t, numChannels, len(channels))
}