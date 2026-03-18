package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

var log = logging.Logger("jala/room")

// Room represents a single GossipSub chat channel.
//
// Design notes:
//   - One Room = one GossipSub topic. Topic name doubles as the room name.
//   - All incoming messages are pushed onto the Messages channel so the
//     UI can consume them without knowing anything about libp2p.
//   - The room is safe to use from multiple goroutines.
type Room struct {
	// Messages is the inbound stream of decoded messages.
	// The UI reads from this channel; the room's read-loop writes to it.
	// Closed when the room is closed.
	Messages <-chan Message

	host   host.Host
	ps     *pubsub.PubSub
	topic  *pubsub.Topic
	sub    *pubsub.Subscription

	roomName string
	selfID   peer.ID
	nick     string

	msgs chan Message
	once sync.Once

	// peerNicks maps PeerID string to nickname.
	// We update this as we see messages from peers.
	peerNicks map[string]string
	nicksMu   sync.RWMutex
}

// Join creates a GossipSub router, joins the topic for roomName, and
// returns a Room ready to send and receive messages.
//
// A join announcement is published immediately so other peers know we
// arrived. ctx is used only for the lifetime of this call; use Close
// to stop the room later.
func Join(
	ctx context.Context,
	h host.Host,
	roomName string,
	nick string,
) (*Room, error) {
	// GossipSub router — one per host is fine; here we create a new one
	// per room for simplicity. In a multi-room app you'd share one router.
	ps, err := pubsub.NewGossipSub(ctx, h,
		pubsub.WithMessageSigning(true),
		pubsub.WithStrictSignatureVerification(true),
		pubsub.WithFloodPublish(true),
	)
	if err != nil {
		return nil, fmt.Errorf("room: gossipsub: %w", err)
	}

	topicName := topicForRoom(roomName)
	topic, err := ps.Join(topicName)
	if err != nil {
		return nil, fmt.Errorf("room: join topic %q: %w", topicName, err)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		_ = topic.Close()
		return nil, fmt.Errorf("room: subscribe %q: %w", topicName, err)
	}

	msgs := make(chan Message, 64)
	r := &Room{
		Messages:  msgs,
		host:      h,
		ps:        ps,
		topic:     topic,
		sub:       sub,
		roomName:  roomName,
		selfID:    h.ID(),
		nick:      nick,
		msgs:      msgs,
		peerNicks: make(map[string]string),
	}

	go r.readLoop(ctx)

	// Announce arrival to the room and periodically rebroadcast identity.
	// This helps peers who join late to learn our nickname without us
	// having to send a chat message.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		// Initial announcement
		if err := r.Publish(NewJoin(h.ID().String(), nick)); err != nil {
			log.Warnf("could not publish join announcement: %v", err)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = r.Publish(NewJoin(h.ID().String(), nick))
			}
		}
	}()

	log.Infof("joined room %q as %q", roomName, nick)
	return r, nil
}

// Publish encodes msg and sends it to all room members via GossipSub.
func (r *Room) Publish(msg Message) error {
	data, err := msg.Encode()
	if err != nil {
		return err
	}
	if err := r.topic.Publish(context.Background(), data); err != nil {
		return fmt.Errorf("room: publish: %w", err)
	}
	return nil
}

// Send is a convenience wrapper that builds and publishes a chat message.
func (r *Room) Send(text string) error {
	return r.Publish(NewChat(r.selfID.String(), r.nick, text))
}

// PeerNicks returns a map of PeerID → short ID for all peers currently
// subscribed to the room's topic (including ourselves).
func (r *Room) PeerNicks() []string {
	peers := r.topic.ListPeers()
	out := make([]string, 0, len(peers)+1)
	out = append(out, r.nick+" (you)")

	r.nicksMu.RLock()
	defer r.nicksMu.RUnlock()

	for _, p := range peers {
		id := p.String()
		if nick, ok := r.peerNicks[id]; ok {
			out = append(out, nick)
		} else {
			short := id
			if len(short) > 12 {
				short = short[:12] + "…"
			}
			out = append(out, short)
		}
	}
	return out
}

// RoomName returns the human-visible room name.
func (r *Room) RoomName() string { return r.roomName }

// Nick returns our current nickname.
func (r *Room) Nick() string { return r.nick }

// SelfID returns our PeerID string.
func (r *Room) SelfID() string { return r.selfID.String() }

// Close sends a leave announcement and unsubscribes from the topic.
// Safe to call multiple times.
func (r *Room) Close() {
	r.once.Do(func() {
		// Best-effort leave notice
		_ = r.Publish(NewLeave(r.selfID.String(), r.nick))
		r.sub.Cancel()
		_ = r.topic.Close()
		close(r.msgs)
		log.Infof("left room %q", r.roomName)
	})
}

// readLoop drains the GossipSub subscription and pushes decoded messages
// onto r.msgs. It stops when ctx is cancelled or the subscription closes.
func (r *Room) readLoop(ctx context.Context) {
	for {
		rawMsg, err := r.sub.Next(ctx)
		if err != nil {
			// ctx cancelled or sub closed — normal shutdown path
			return
		}

		// GossipSub echoes our own publishes back; skip them.
		if rawMsg.ReceivedFrom == r.selfID {
			continue
		}

		msg, err := Decode(rawMsg.Data)
		if err != nil {
			log.Debugf("malformed message from %s: %v", rawMsg.ReceivedFrom, err)
			continue
		}

		// Update nickname map
		r.nicksMu.Lock()
		oldNick, exists := r.peerNicks[msg.SenderID]
		isNew := !exists || (msg.Type == MsgJoin && oldNick != msg.Nick)
		if msg.Type == MsgLeave {
			delete(r.peerNicks, msg.SenderID)
		} else {
			r.peerNicks[msg.SenderID] = msg.Nick
		}
		r.nicksMu.Unlock()

		// For MsgJoin, only push to UI if it's actually new/changed to avoid
		// UI spam from periodic rebroadcasts.
		if msg.Type == MsgJoin && !isNew {
			continue
		}

		select {
		case r.msgs <- msg:
		case <-ctx.Done():
			return
		default:
			// Drop if the UI is not consuming fast enough; never block network.
			log.Warn("message buffer full — dropping message")
		}
	}
}

// topicForRoom returns the GossipSub topic string for a room name.
// Namespacing prevents collisions with other libp2p applications.
func topicForRoom(name string) string {
	return "/jala/room/v1/" + name
}
