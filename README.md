# Go Chat Server

A modern, secure chat server built with Go, featuring JWT authentication, comprehensive channel management, and full audit logging capabilities.

## Features

- **🔐 Secure Authentication**: JWT-based authentication with refresh tokens and bcrypt password hashing
- **👥 User Management**: Complete user lifecycle with account updates and deletion
- **💬 Channel System**: Create, join, and manage channels with password protection and visibility controls
- **🛡️ Admin Controls**: User banning, role management, and channel administration
- **🔍 Search Functionality**: Search users, channels, and message history with filtering
- **📋 Audit Logging**: Comprehensive audit trails for all admin actions with metadata
- **💾 Message History**: Configurable message logging and retrieval system
- **⚡ Rate Limiting**: Tiered IP-based rate limiting to protect against abuse
- **📚 API Documentation**: Complete Swagger/OpenAPI documentation with interactive UI
- **🔒 TLS Security**: HTTPS-only server with certificate-based encryption
- **💾 SQLite Database**: Lightweight database with GORM ORM and auto-migrations
- **🧪 Test Coverage**: Test-driven development with comprehensive test suites

## Quick Start

### Prerequisites

- Go 1.21 or later
- Make (optional, for convenience commands)
- OpenSSL (for certificate generation)

### Installation

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd go-chat
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Generate required certificates and secrets:**
   ```bash
   make generate-cert    # Creates cert.pem and key.pem
   make generate-secret  # Creates .env with APP_SECRET
   ```

4. **Run the server:**
   ```bash
   go run ./cmd/server
   ```

The server will start on `https://localhost:9876`

### Troubleshooting

**TLS Certificate Warnings:**
When accessing `https://localhost:9876`, your browser will show a security warning because we're using a self-signed certificate. This is normal for development:

- **Chrome/Edge**: Click "Advanced" → "Proceed to localhost (unsafe)"
- **Firefox**: Click "Advanced" → "Accept the Risk and Continue" 
- **Safari**: Click "Show Details" → "visit this website"

**Alternative: Use curl to test:**
```bash
# Health check
curl -k https://localhost:9876/hc

# API Documentation (JSON)
curl -k https://localhost:9876/swagger/doc.json
```

**Port Already in Use:**
```bash
# Find process using port 9876
ss -tulpn | grep :9876

# Kill the process (replace PID with actual process ID)
kill -9 <PID>
```

## API Documentation

Once the server is running, visit `https://localhost:9876/swagger/index.html` for interactive API documentation.

### Key Endpoints

#### Authentication
- `POST /register` - Register a new user
- `POST /login` - User login
- `POST /api/logout` - Logout (requires auth)
- `POST /api/refresh_token` - Refresh JWT token

#### User Management
- `PATCH /api/user` - Update username/password
- `DELETE /api/user` - Delete account
- `GET /api/user/channels/owned` - List owned channels
- `GET /api/user/channels/joined` - List joined channels

#### Channels
- `GET /api/channels` - List all visible channels
- `POST /api/channels` - Create a new channel
- `GET /api/channels/:id` - Get channel details
- `GET /api/channels/:id/users` - List channel members
- `POST /api/channels/:id/join` - Join a channel
- `DELETE /api/channels/:id/leave` - Leave a channel
- `DELETE /api/channels/:id` - Delete channel (owner only)

#### Channel Administration
- `POST /api/channels/:id/ban` - Permanently ban a user
- `POST /api/channels/:id/tempban` - Temporarily ban a user
- `DELETE /api/channels/:id/ban/:userId` - Unban a user
- `GET /api/channels/:id/bans` - List channel bans
- `POST /api/channels/:id/promote` - Promote user role
- `POST /api/channels/:id/demote` - Demote user role

#### Message History
- `GET /api/channels/:id/messages` - Get channel message history

#### Search
- `GET /api/search/users` - Search users by username
- `GET /api/search/channels` - Search visible channels by name
- `GET /api/search/messages` - Search messages within a channel

#### Audit Logs
- `GET /api/channels/:id/audit` - Channel audit logs (owner only)
- `GET /api/audit` - System audit logs with filtering

## Rate Limiting

The server implements tiered rate limiting:

- **Authentication endpoints**: 5 req/sec (burst: 10) - Strict protection against brute force
- **General API endpoints**: 30 req/sec (burst: 50) - Standard protection
- **Read-only endpoints**: 100 req/sec (burst: 200) - Lenient for browsing

## Development

### Build Commands

```bash
make build-linux      # Build for Linux (amd64)
make build-macos      # Build for macOS (arm64) 
make build-windows    # Build for Windows (amd64)
make clear            # Clean build artifacts
```

### Testing

```bash
make test             # Run all tests
make test-verbose     # Run with verbose output
make test-coverage    # Generate coverage report
```

**Run specific test suites:**
```bash
go test ./internal/api/...       # Test API layer
go test ./internal/audit/...     # Test audit system
go test ./internal/auth/...      # Test authentication
go test -v ./internal/search/... # Test search with verbose output
```

### Documentation

```bash
make docs             # Generate Swagger documentation
```

### Database

The server uses SQLite with automatic migrations. The database file `gochat.db` is created automatically on first run.

**Default roles seeded:**
- Administrator - Full system access
- Moderator - Channel moderation capabilities
- Member - Standard user privileges  
- Guest - Limited read-only access

## Architecture

### Project Structure

```
cmd/server/          # Application entry point
internal/
  api/               # HTTP handlers and routing
  audit/             # Audit logging system
  auth/              # Authentication middleware and logic
  channel/           # Channel business logic
  message/           # Message management
  middleware/        # HTTP middleware (rate limiting, etc.)
  search/            # Search functionality
  storage/           # Database configuration
  user/              # User management business logic
  utils/             # Shared utilities
  version/           # Version information
pkg/chat/            # Shared data models
docs/                # Generated API documentation
```

### Technology Stack

- **Framework**: Gin (HTTP), Gorilla WebSocket
- **Database**: SQLite with GORM ORM
- **Authentication**: JWT with golang-jwt/jwt/v5
- **Security**: bcrypt, TLS/HTTPS, rate limiting
- **Documentation**: Swagger/OpenAPI with swaggo/swag
- **ID Generation**: Custom nanoid implementation
- **Testing**: testify/assert for comprehensive test suites

## Configuration

### Environment Variables

Create a `.env` file (or use `make generate-secret`):

```env
APP_SECRET=your-jwt-signing-secret-here
```

### TLS Certificates

Generate certificates (or use `make generate-cert`):

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

**Example certificate generation:**
```bash
# When prompted, you can use these example values:
# Country Name: US
# State: California  
# City: San Francisco
# Organization: Dev
# Organizational Unit: IT
# Common Name: localhost
# Email: dev@localhost
```

## Usage Examples

### Register and Login
```bash
# Register a new user
curl -k -X POST https://localhost:9876/register \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "password": "securepass123"}'

# Login
curl -k -X POST https://localhost:9876/login \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "password": "securepass123"}'
```

### Create and Join Channels
```bash
# Create a channel (requires auth cookie from login)
curl -k -X POST https://localhost:9876/api/channels \
  -H "Content-Type: application/json" \
  -b "cookies.txt" \
  -d '{"name": "general", "is_visible": true}'

# Join a channel
curl -k -X POST https://localhost:9876/api/channels/{channel-id}/join \
  -H "Content-Type: application/json" \
  -b "cookies.txt" \
  -d '{}'
```

## Security Features

- **JWT Authentication**: Secure token-based authentication with refresh tokens
- **Password Hashing**: bcrypt with salt for secure password storage
- **Rate Limiting**: Protection against brute force and spam attacks
- **TLS Encryption**: All traffic encrypted with HTTPS
- **Input Validation**: Comprehensive request validation and sanitization
- **Audit Logging**: Complete audit trail of all admin actions
- **Access Controls**: Role-based permissions for channel management

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and development process.

## Health Check

The server provides a health check endpoint at `GET /hc` that returns "Running" when the server is operational.

```bash
curl -k https://localhost:9876/hc
# Response: Running
```

## Architecture Diagram

### System Overview

```mermaid
graph TB
    subgraph "Client Layer"
        Browser[Web Browser/Client]
        Curl[curl/API Client]
    end
    
    subgraph "TLS/Security"
        TLS[TLS Certificate<br/>cert.pem/key.pem]
    end
    
    subgraph "Go Chat Server :9876"
        subgraph "HTTP Layer"
            Gin[Gin Router<br/>Rate Limited]
            Auth[Auth Middleware<br/>JWT Validation]
            Routes[API Routes<br/>26+ endpoints]
        end
        
        subgraph "API Handlers"
            AuthH[Auth Handlers<br/>login/register]
            UserH[User Handlers<br/>CRUD operations]
            ChannelH[Channel Handlers<br/>management/admin]
            MessageH[Message Handlers<br/>history/retrieval]
            SearchH[Search Handlers<br/>users/channels/messages]
            AuditH[Audit Handlers<br/>logs/compliance]
        end
        
        subgraph "Business Logic"
            AuthS[Auth Service<br/>JWT/bcrypt]
            UserS[User Service<br/>account mgmt]
            ChannelS[Channel Service<br/>with audit logging]
            MessageS[Message Service<br/>history mgmt]
            SearchS[Search Service<br/>with filters]
            AuditS[Audit Service<br/>comprehensive logging]
        end
        
        subgraph "Data Layer"
            GORM[GORM ORM<br/>Auto-migration]
            SQLite[(SQLite DB<br/>gochat.db)]
        end
        
        subgraph "Documentation"
            Swagger[Swagger UI<br/>/swagger/index.html]
            OpenAPI[OpenAPI Spec<br/>docs/]
        end
    end
    
    subgraph "Configuration"
        Env[.env file<br/>APP_SECRET]
        Config[TLS Config<br/>HTTPS Only]
    end
    
    %% Client connections
    Browser -->|HTTPS Requests| TLS
    Curl -->|HTTPS + -k flag| TLS
    TLS -->|Encrypted Traffic| Gin
    
    %% HTTP Flow
    Gin -->|Authentication| Auth
    Auth -->|Validated Requests| Routes
    Routes -->|Route to Handler| AuthH
    Routes -->|Route to Handler| UserH
    Routes -->|Route to Handler| ChannelH
    Routes -->|Route to Handler| MessageH
    Routes -->|Route to Handler| SearchH
    Routes -->|Route to Handler| AuditH
    
    %% Handler to Service mapping
    AuthH -->|Business Logic| AuthS
    UserH -->|Business Logic| UserS
    ChannelH -->|Business Logic| ChannelS
    MessageH -->|Business Logic| MessageS
    SearchH -->|Business Logic| SearchS
    AuditH -->|Business Logic| AuditS
    
    %% Service to Data mapping
    AuthS -->|Data Access| GORM
    UserS -->|Data Access| GORM
    ChannelS -->|Data Access + Audit| GORM
    ChannelS -->|Automatic Logging| AuditS
    MessageS -->|Data Access| GORM
    SearchS -->|Data Queries| GORM
    AuditS -->|Audit Storage| GORM
    GORM -->|ORM Operations| SQLite
    
    %% Documentation
    Routes -->|API Specs| Swagger
    Swagger -->|Generated Docs| OpenAPI
    
    %% Configuration
    Env -->|JWT Secret| AuthS
    Config -->|TLS Setup| TLS
    
    classDef handler fill:#e1f5fe
    classDef service fill:#f3e5f5
    classDef data fill:#e8f5e8
    classDef security fill:#fff3e0
    classDef client fill:#fce4ec
    
    class AuthH,UserH,ChannelH,MessageH,SearchH,AuditH handler
    class AuthS,UserS,ChannelS,MessageS,SearchS,AuditS service
    class GORM,SQLite data
    class TLS,Auth,Env security
    class Browser,Curl client
```

### Data Flow for Admin Actions

```mermaid
sequenceDiagram
    participant Client
    participant Auth as Auth Middleware
    participant ChannelH as Channel Handler
    participant ChannelS as Channel Service
    participant AuditS as Audit Service
    participant DB as SQLite Database
    
    Client->>+Auth: POST /api/channels/123/ban
    Auth->>Auth: Validate JWT Token
    Auth->>+ChannelH: Forward Request
    
    ChannelH->>+ChannelS: BanUser(adminID, userID, channelID, reason)
    ChannelS->>DB: Check permissions & validate
    ChannelS->>DB: Create ban record
    ChannelS->>+AuditS: LogUserBan(adminID, userID, channelID, reason, false, nil)
    AuditS->>DB: Store audit log with metadata
    AuditS-->>-ChannelS: Audit logged
    ChannelS->>DB: Remove user from channel
    ChannelS-->>-ChannelH: Ban successful
    
    ChannelH-->>-Auth: Success response
    Auth-->>-Client: 200 OK {"message": "User banned successfully"}
    
    Note over AuditS,DB: Audit log includes:<br/>- Action: BAN_USER<br/>- Actor, Target, Channel<br/>- Reason in metadata<br/>- Timestamp
```

### Database Schema Overview

```mermaid
erDiagram
    Users ||--o{ UserChannels : "belongs to"
    Users ||--o{ AuditLogs : "performs actions"
    Users ||--o{ Messages : "writes"
    Users ||--o{ UserBans : "can be banned"
    Users ||--o{ UserIPs : "has IPs"
    Users ||--o{ RefreshTokens : "has tokens"
    
    Channels ||--o{ UserChannels : "contains"
    Channels ||--o{ Messages : "stores"
    Channels ||--o{ UserBans : "has bans"
    Channels ||--o{ AuditLogs : "tracked in"
    
    Roles ||--o{ UserChannels : "assigned to"
    
    Users {
        string id PK "nanoid(8)"
        string username UK "unique"
        string password "bcrypt hash"
        timestamp created_at
        timestamp updated_at
    }
    
    Channels {
        string id PK "nanoid(6)"
        string name UK "unique"
        boolean is_visible
        string password "optional"
        uint logging_days "message retention"
        string owner_id FK
        timestamp created_at
    }
    
    UserChannels {
        uint id PK
        string user_id FK
        string channel_id FK
        uint role_id FK
        timestamp joined_at
    }
    
    AuditLogs {
        uint id PK
        string action "BAN_USER, PROMOTE_USER, etc"
        string actor_id FK
        string target_id FK "optional"
        string channel_id FK "optional"
        string description "human readable"
        json metadata "structured data"
        timestamp created_at
    }
    
    Messages {
        string id PK "nanoid(10)"
        string content
        string user_id FK
        string channel_id FK
        timestamp created_at
    }
    
    UserBans {
        uint id PK
        string user_id FK
        string channel_id FK
        string banned_by FK
        string reason
        timestamp expires_at "null = permanent"
        boolean is_active
        timestamp created_at
    }
    
    Roles {
        uint id PK
        string name UK "Administrator, Moderator, etc"
    }
```

### Rate Limiting Strategy

```mermaid
graph LR
    subgraph "Rate Limiting Tiers"
        subgraph "Strict (5 req/sec)"
            A1[POST /register]
            A2[POST /login]
        end
        
        subgraph "Standard (30 req/sec)"
            B1[POST /api/channels]
            B2[POST /api/channels/:id/ban]
            B3[PATCH /api/user]
            B4[DELETE /api/user]
        end
        
        subgraph "Lenient (100 req/sec)"
            C1[GET /api/channels]
            C2[GET /api/search/*]
            C3[GET /api/audit]
            C4[GET /api/channels/:id/audit]
            C5[GET /hc]
        end
    end
    
    Client[Client Request] --> IPCheck{IP Address}
    IPCheck --> RateLimit{Rate Limit Check}
    RateLimit -->|Within Limit| Allow[Process Request]
    RateLimit -->|Exceeded| Block[429 Too Many Requests]
    
    Allow --> Auth[Authentication Check]
    Auth -->|Valid| Handler[Route to Handler]
    Auth -->|Invalid| Reject[401 Unauthorized]
```

## License

This project is licensed under the GNU Affero General Public License v3.0 - see the LICENSE file for details.