## jala — P2P terminal chat
## Requires: go 1.22+

BINARY  := jala
CMD     := ./cmd/jala
LDFLAGS := -trimpath -ldflags="-s -w"

.PHONY: all build run run-debug test test-race lint tidy clean help

all: build

## build: compile to ./bin/jala
build:
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY) $(CMD)
	@echo "✓ bin/$(BINARY)"

## run: start a node (ARGS="..." for extra flags)
run:
	go run $(CMD) $(ARGS)

## room: join a specific room, e.g. make room ROOM=dev
room:
	go run $(CMD) -room $(ROOM) $(ARGS)

## run-debug: run with verbose logging to jala-debug.log
run-debug:
	go run $(CMD) -debug $(ARGS)

## two: run two nodes locally (for testing mDNS discovery)
two:
	@echo "Starting node 1 on port 14001…"
	go run $(CMD) -port 14001 -nick peerOne -no-dht &
	@sleep 1
	@echo "Starting node 2 on port 14002…"
	go run $(CMD) -port 14002 -nick peerTwo -no-dht

## test: run all unit tests
test:
	go test ./... -v -count=1

## test-race: run with race detector
test-race:
	go test -race ./... -v -count=1

## tidy: update go.mod and go.sum
tidy:
	go mod tidy

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## clean: remove build artifacts
clean:
	rm -rf bin/ jala-debug.log

## help: show available targets
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
