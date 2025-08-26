VERSION := $(shell git describe --tags --always)
COMMIT  := $(shell git rev-parse --short HEAD)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X 'gochat/internal/version.Version=$(VERSION)' \
           -X 'gochat/internal/version.Commit=$(COMMIT)' \
           -X 'gochat/internal/version.BuildTime=$(DATE)'


build-macos:
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/gochat-client-macos ./cmd/client 
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/gochat-server-macos ./cmd/server 

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/gochat-client-linux ./cmd/client
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/gochat-server-linux ./cmd/server

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/gochat-client.exe ./cmd/client
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/gochat-server.exe ./cmd/server

generate-cert:
	openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

generate-secret:
	echo "APP_SECRET=$$(openssl rand -hex 32)" > .env

clear:
	rm -f bin/*
