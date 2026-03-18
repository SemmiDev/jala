// Command jala is a terminal P2P chat application built on go-libp2p.
//
// Usage:
//
//	jala [flags]
//
// Flags:
//
//	-nick    your display name (default: random adjective-animal)
//	-room    chat room name   (default: "lobby")
//	-key     path to persist Ed25519 private key (default: ephemeral)
//	-port    TCP/UDP listen port (default: OS-assigned)
//	-no-mdns disable LAN discovery
//	-no-dht  disable global DHT discovery
//	-debug   enable verbose logging
package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"

	logging "github.com/ipfs/go-log/v2"
	"github.com/semmidev/jala/internal/chat"
	"github.com/semmidev/jala/internal/node"
	"github.com/semmidev/jala/internal/ui"
	"github.com/semmidev/jala/pkg/identity"
)

func main() {
	// ── Flags ────────────────────────────────────────────────────────────
	nick := flag.String("nick", "", "display name (default: random)")
	room := flag.String("room", "lobby", "chat room name")
	keyPath := flag.String("key", "", "Ed25519 key file for stable identity")
	port := flag.Int("port", 0, "listen port (0 = random)")
	noMDNS := flag.Bool("no-mdns", false, "disable LAN peer discovery")
	noDHT := flag.Bool("no-dht", false, "disable global DHT discovery")
	debug := flag.Bool("debug", false, "verbose logging")
	flag.Parse()

	// ── Logging ──────────────────────────────────────────────────────────
	// In TUI mode we silence all logs so they don't corrupt the screen.
	// Pass -debug to write logs to jala-debug.log instead.
	logLevel := "error"
	if *debug {
		logLevel = "debug"
	}
	if err := logging.SetLogLevel("*", logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "log level error: %v\n", err)
	}
	if *debug {
		// Write logs to a file so the TUI stays clean
		f, err := os.OpenFile("jala-debug.log",
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err == nil {
			logging.SetupLogging(logging.Config{
				File:  "jala-debug.log",
				Level: logging.LevelDebug,
			})
			defer f.Close()
		}
	}

	// ── Identity ─────────────────────────────────────────────────────────
	id, err := identity.Load(*keyPath)
	if err != nil {
		fatal("load identity: %v", err)
	}

	// ── Nickname ─────────────────────────────────────────────────────────
	displayNick := *nick
	if displayNick == "" {
		displayNick = randomNick()
	}

	// ── Context & shutdown ────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── Build the libp2p node ─────────────────────────────────────────────
	nodeCfg := node.DefaultConfig(id.PrivKey)
	nodeCfg.EnableMDNS = !*noMDNS
	nodeCfg.EnableDHT = !*noDHT
	if *port != 0 {
		nodeCfg.ListenAddrs = []string{
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port),
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", *port),
		}
	}

	n, err := node.New(ctx, nodeCfg)
	if err != nil {
		fatal("start node: %v", err)
	}
	defer n.Close()

	// ── Join the chat room ────────────────────────────────────────────────
	r, err := chat.Join(ctx, n.Host, *room, displayNick)
	if err != nil {
		fatal("join room: %v", err)
	}
	defer r.Close()

	// ── Launch TUI ───────────────────────────────────────────────────────
	// ui.Run blocks until the user presses Esc / Ctrl-C or ctx is cancelled.
	if err := ui.Run(ctx, r); err != nil {
		fatal("ui: %v", err)
	}
}

// fatal prints msg to stderr and exits 1.
func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "jala: "+format+"\n", args...)
	os.Exit(1)
}

// ── Nickname generator ────────────────────────────────────────────────────────

var (
	adjectives = []string{
		"ancient", "bold", "calm", "daring", "eager",
		"fierce", "gentle", "hidden", "idle", "jolly",
		"keen", "lively", "mystic", "nimble", "odd",
		"proud", "quiet", "rapid", "silent", "tough",
		"urban", "vivid", "wild", "young", "zealous",
	}
	animals = []string{
		"badger", "crane", "dingo", "eagle", "falcon",
		"gecko", "heron", "ibis", "jackal", "kestrel",
		"lemur", "marten", "newt", "otter", "panther",
		"quoll", "raven", "stoat", "tapir", "urial",
		"viper", "walrus", "xerus", "yak", "zorilla",
	}
)

// randomNick returns a memorable adjective-animal nickname, e.g. "silentOtter".
func randomNick() string {
	adj := adjectives[rand.Intn(len(adjectives))]
	animal := animals[rand.Intn(len(animals))]
	// capitalise the animal part for camelCase readability
	return adj + string(animal[0]-32) + animal[1:]
}
