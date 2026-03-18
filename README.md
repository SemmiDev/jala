# jala

> **jala** — bahasa Indonesia untuk *jaring* (net/mesh)

Terminal P2P chat built on [go-libp2p](https://github.com/libp2p/go-libp2p)
and [Bubble Tea](https://github.com/charmbracelet/bubbletea).
No server. No account. Just run it.

```
┌─ jala #lobby ──────────────────────────────────────────┐
│                                                          │
│  --- silentOtter joined ---                              │
│  --- boldFalcon joined ---                               │
│  15:04 silentOtter: hey, anyone here?                    │
│  15:04 boldFalcon: yep! p2p works                        │
│  15:05 silentOtter: nice, no server needed               │
│                                                          │
│> _                                                       │
├──────────────────────────────────────────────────────────┤
│ #lobby  nick: silentOtter  id: 12D3KooWNze…      15:05  │
│ Enter send  Esc quit  Tab peers  Ctrl+L clear            │
└──────────────────────────────────────────────────────────┘
```

## Quick Start

```bash
git clone https://github.com/yourname/jala
cd jala
go mod tidy
go run ./cmd/jala
```

Two nodes on the same LAN will find each other automatically via mDNS within
seconds. Two nodes on different networks connect via the IPFS DHT.

## Usage

```
jala [flags]

  -nick   string   display name (default: random, e.g. "silentOtter")
  -room   string   chat room    (default: "lobby")
  -key    string   path to Ed25519 key file for stable identity
  -port   int      TCP/UDP listen port (default: random)
  -no-mdns         disable LAN peer discovery
  -no-dht          disable global DHT discovery
  -debug           write verbose logs to jala-debug.log
```

### Examples

```bash
# Join the default lobby with a random nickname
go run ./cmd/jala

# Join a room called "dev" with a specific nickname
go run ./cmd/jala -room dev -nick alice

# Stable identity across restarts
go run ./cmd/jala -key ~/.jala-identity

# Two nodes on the same machine (for testing)
make two
```

## Keyboard Shortcuts

| Key      | Action                          |
|----------|---------------------------------|
| `Enter`  | Send message                    |
| `Esc`    | Quit                            |
| `Tab`    | Toggle peer list panel          |
| `Ctrl+L` | Clear chat history              |
| `Ctrl+C` | Force quit                      |

## Architecture

```
cmd/jala/main.go
    │
    ├── pkg/identity        Ed25519 key load/generate/persist
    │
    ├── internal/node       libp2p host + mDNS + Kademlia DHT
    │       │
    │       └── host.Host   TCP+QUIC transports
    │                       Noise+TLS security
    │                       ConnManager (20–100 peers)
    │                       AutoNAT + Relay + HolePunching
    │
    ├── internal/chat
    │       ├── message.go  Wire format (JSON), MsgType enum
    │       └── room.go     GossipSub topic join/publish/subscribe
    │
    └── internal/ui
            ├── app.go      Bubble Tea root model (Elm Architecture)
            ├── chatview.go Message history panel + peer list
            └── styles.go   Lipgloss colour palette
```

### Why GossipSub for group chat?

GossipSub maintains a mesh of D=6 peers per topic and propagates
messages via epidemic broadcast — O(log n) hops instead of O(n)
for FloodSub. Every subscriber receives every message with no
central broker.

### Why Bubble Tea for the UI?

Bubble Tea's **Elm Architecture** (Model → Update → View) keeps
the UI single-threaded and race-condition-free. Network events
(incoming messages, peer changes) are delivered as typed `Msg`
values through the event loop, never mutating state directly.

### Peer Discovery Flow

```
LAN:    mDNS multicast → HandlePeerFound → host.Connect
Global: DHT bootstrap → Advertise "/jala/rendezvous/v1"
                      → FindPeers every 10s → host.Connect
```

## Development

```bash
make test        # unit tests
make test-race   # with race detector
make lint        # golangci-lint
make build       # compile to ./bin/jala
```
