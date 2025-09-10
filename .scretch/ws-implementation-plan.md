# WebSocket Implementation Plan

## Overview
This document outlines the step-by-step implementation plan for the WebSocket real-time chat functionality. The architecture and test framework are already in place - this plan focuses on implementing the actual functionality following TDD principles.

## Phase 1: Core WebSocket Infrastructure

### 1.1 Hub Implementation (`internal/websocket/hub.go`)
- [ ] **Implement `Run()` method**
  - Handle register/unregister channels in goroutine loop
  - Process broadcast messages from channels
  - Clean up disconnected clients
  - Handle graceful shutdown

- [ ] **Implement client registration**
  - `RegisterClient(client *Client)` - add client to hub maps
  - Thread-safe client tracking
  - Initialize client in channels map

- [ ] **Implement client unregistration** 
  - `UnregisterClient(client *Client)` - remove from all data structures
  - Clean up client from all channels
  - Close client channels safely

- [ ] **Implement channel management**
  - `JoinChannel(client *Client, channelID string)` - add client to channel
  - `LeaveChannel(client *Client, channelID string)` - remove from channel
  - `GetChannelClients(channelID string)` - return client list
  - `GetClientByUserID(userID string)` - find client by user ID

- [ ] **Implement broadcasting**
  - `BroadcastToChannel()` - send message to all channel clients
  - `BroadcastToUser()` - send message to specific user
  - Handle client exclusion in broadcasts
  - Ensure thread safety

- [ ] **Implement utility methods**
  - `GetClientCount()`, `GetChannelClientCount()`
  - `IsClientInChannel()` validation
  - Null pointer safety checks

### 1.2 Client Implementation (`internal/websocket/client.go`)

- [ ] **Implement `SendMessage()` method**
  - JSON marshal WebSocket message
  - Write to client's send channel
  - Handle send channel blocking/full scenarios
  - Error handling and logging

- [ ] **Implement `ReadPump()` method**
  - Set up WebSocket read loop in goroutine
  - Handle ping/pong for connection health
  - Set read deadlines and limits
  - Parse incoming messages and delegate to MessageHandler
  - Handle connection errors and cleanup

- [ ] **Implement `WritePump()` method** 
  - Set up WebSocket write loop in goroutine
  - Read from send channel and write to WebSocket
  - Handle write deadlines and timeouts
  - Send periodic ping messages
  - Handle connection errors and cleanup

- [ ] **Implement `Close()` method**
  - Close WebSocket connection gracefully
  - Close send channel
  - Cleanup resources
  - Notify hub of disconnection

## Phase 2: Message Processing

### 2.1 Message Handler Implementation (`internal/websocket/message_handler.go`)

- [ ] **Implement `HandleMessage()` method**
  - Parse JSON message from client
  - Route to appropriate handler based on message type
  - Handle JSON parsing errors
  - Update client last seen timestamp

- [ ] **Implement chat message handling**
  - `handleChatMessage()` - process chat messages
  - Validate channel access permissions
  - Save message to database using existing models
  - Broadcast message to channel clients
  - Handle message content validation/sanitization

- [ ] **Implement channel management**
  - `handleJoinChannel()` - validate and join user to channel
  - `handleLeaveChannel()` - remove user from channel
  - Check channel permissions and bans
  - Broadcast user join/leave events
  - Update client and hub channel memberships

- [ ] **Implement typing indicators**
  - `handleTyping()` - process typing start/stop
  - Validate channel membership
  - Broadcast typing status to other channel members
  - Implement typing timeout logic

- [ ] **Implement ping/pong**
  - `handlePing()` - respond with pong message
  - Update connection health tracking

- [ ] **Implement utility methods**
  - `sendErrorToClient()` - send formatted error responses
  - `validateChannelAccess()` - check user permissions
  - `saveMessage()` - persist chat messages
  - Message broadcasting helpers

### 2.2 Database Integration

- [ ] **Integrate with existing Message model**
  - Use existing `chat.Message` struct for persistence
  - Generate message IDs using nanoid
  - Handle database transaction errors

- [ ] **Integrate with Channel service**
  - Use existing channel permission checking
  - Validate user membership and bans
  - Handle private/password-protected channels

- [ ] **Integrate with Audit service**
  - Log WebSocket connection events
  - Log channel join/leave events  
  - Log message sending for audit trail

## Phase 3: HTTP WebSocket Handler

### 3.1 WebSocket Upgrade Handler (`internal/api/websocket.go`)

- [ ] **Implement `HandleWebSocket()` method**
  - Extract JWT token from request (header or cookie)
  - Validate authentication using existing middleware
  - Upgrade HTTP connection to WebSocket
  - Create new Client instance
  - Register client with hub
  - Start client read/write pumps
  - Handle upgrade errors

- [ ] **Implement authentication helpers**
  - `authenticateWebSocket()` - extract and validate JWT
  - Handle token expiration and refresh
  - Extract user ID and username from claims

- [ ] **Implement connection management**
  - `handleClientConnection()` - setup new client
  - `handleClientDisconnection()` - cleanup on disconnect
  - `authorizeChannelAccess()` - check channel permissions
  - `logWebSocketEvent()` - audit logging

### 3.2 WebSocket Info Endpoints

- [ ] **Implement `GetConnectionInfo()` method**
  - Return total connection count
  - Return per-channel statistics
  - Return active user list with metadata
  - Require admin permissions

- [ ] **Implement `GetChannelStats()` method**
  - Return channel-specific connection info
  - List connected users in channel
  - Calculate recent activity metrics
  - Validate channel access permissions

## Phase 4: Integration & Testing

### 4.1 End-to-End Integration

- [ ] **Run existing test suites**
  - Ensure all WebSocket tests pass
  - Verify no regressions in existing functionality
  - Test concurrent client connections

- [ ] **Integration testing**
  - Test with actual WebSocket clients
  - Verify message flow between multiple clients
  - Test channel subscription/unsubscription
  - Test authentication and authorization

- [ ] **Performance testing**
  - Test with multiple concurrent connections
  - Measure message latency and throughput
  - Test hub scalability with many channels
  - Monitor memory usage and connection cleanup

### 4.2 Error Handling & Edge Cases

- [ ] **Connection error handling**
  - Test client disconnections during message sending
  - Handle network timeouts and reconnections
  - Test WebSocket upgrade failures
  - Handle malformed message formats

- [ ] **Security testing**
  - Test unauthorized access attempts
  - Verify JWT token validation
  - Test channel permission enforcement
  - Test message content validation

## Phase 5: Documentation & Deployment

### 5.1 Update Documentation

- [ ] **Update API documentation**
  - Document WebSocket endpoints in Swagger
  - Update message protocol documentation
  - Add WebSocket connection examples

- [ ] **Update README.md**
  - Add WebSocket feature description
  - Update architecture diagrams
  - Add WebSocket client connection examples
  - Document WebSocket message protocol

### 5.2 Production Readiness

- [ ] **Configuration management**
  - Add WebSocket-specific config options
  - Configure connection limits and timeouts
  - Set up monitoring and logging

- [ ] **Deployment considerations**
  - Test WebSocket functionality with reverse proxy
  - Verify HTTPS WebSocket connections
  - Test load balancing with sticky sessions if needed

## Implementation Notes

### Key Principles
1. **Follow TDD**: Run tests after each implementation step
2. **Thread Safety**: Use mutexes and channels appropriately  
3. **Error Handling**: Always handle and log errors gracefully
4. **Resource Cleanup**: Ensure proper connection and goroutine cleanup
5. **Security**: Validate all inputs and enforce permissions

### Testing Strategy
1. **Unit Tests**: Test each method individually with mocks
2. **Integration Tests**: Test component interactions
3. **Concurrent Tests**: Test thread safety and race conditions
4. **E2E Tests**: Test full WebSocket message flow

### Performance Considerations
1. **Connection Limits**: Set reasonable max connection limits
2. **Message Queuing**: Handle slow consumers with buffered channels
3. **Memory Management**: Clean up disconnected clients promptly
4. **Broadcasting Efficiency**: Optimize message distribution

## Dependencies Already Available
- Gorilla WebSocket library
- JWT authentication system
- Database models and services
- Audit logging system
- Channel permission system
- Test framework and mocks

This implementation plan provides a structured approach to building the WebSocket functionality while maintaining code quality and following established patterns in the codebase.