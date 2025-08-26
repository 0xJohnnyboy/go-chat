# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

### Build Commands
- `make build-linux` - Build for Linux (amd64)
- `make build-macos` - Build for macOS (arm64) 
- `make build-windows` - Build for Windows (amd64)
- `make clear` - Clean build artifacts from bin/ directory

### Server Setup
- `make generate-cert` - Generate TLS certificate files (cert.pem, key.pem) for HTTPS
- `make generate-secret` - Generate APP_SECRET environment variable in .env file

### Running the Server
- `go run ./cmd/server` - Run the server locally (default port :9876)
- The server runs on HTTPS using cert.pem/key.pem certificates

## Architecture Overview

### Core Structure
This is a Go-based chat server with JWT authentication, WebSocket support, and SQLite database storage using GORM ORM.

**Module name**: `go-chat` (as defined in go.mod)

### Key Components

#### 1. Database Layer (`internal/storage/`)
- Uses GORM with SQLite driver
- Database file: `gochat.db` (auto-created)
- Auto-migration on startup
- Seeds default roles: Administrator, Moderator, Member, Guest

#### 2. Authentication System (`internal/auth/`)
- JWT-based authentication with refresh tokens
- Password hashing using bcrypt
- Role-based access control
- IP tracking for users

#### 3. API Layer (`internal/api/`)
- Gin framework for HTTP routing
- TLS-only server (requires cert.pem/key.pem)
- Separate route groups for protected/unprotected endpoints

#### 4. Data Models (`pkg/chat/`)
- User management with nanoid-based IDs (8 chars for users, 6 for channels)
- Channel system with visibility and password protection
- User-Channel relationships with roles
- Refresh token management

### Key Dependencies
- **Gin**: HTTP web framework
- **GORM**: ORM with SQLite driver  
- **Gorilla WebSocket**: WebSocket implementation
- **JWT**: golang-jwt/jwt/v5 for token handling
- **bcrypt**: golang.org/x/crypto for password hashing
- **nanoid**: Custom ID generation

### Database Schema
- **Users**: ID (nanoid), Username, Password (hashed), timestamps
- **Channels**: ID (nanoid), Name (unique), visibility, optional password, owner
- **UserChannels**: Many-to-many relationship with roles
- **RefreshTokens**: Token management with expiration
- **UserIPs**: IP address tracking
- **Roles**: Seeded role system

### Environment and Security
- Requires .env file with APP_SECRET for JWT signing
- TLS certificates (cert.pem, key.pem) required for HTTPS
- Database and certificates are gitignored
- Build uses ldflags for version injection from git tags/commits

### Current State
- Basic authentication endpoints implemented (/register, /login, /logout, /refresh_token)
- Database models and relationships defined
- WebSocket infrastructure partially implemented
- No test suite currently exists