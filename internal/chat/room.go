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
type Room struct {
	// Messages is the inbound stream of decoded messages.
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
	peerNicks map[string]string
	nicksMu   sync.RWMutex
}

// Join joins the topic for roomName on the given GossipSub router and
// returns a Room ready to send and receive messages.
func Join(
	ctx context.Context,
	h host.Host,
	ps *pubsub.PubSub,
	roomName string,
	nick string,
) (*Room, error) {
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
	go func() {
		// Wait a small bit for GossipSub to find peers before announcing.
		// If we announce TOO fast, nobody is listening yet.
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return
		}

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

// PeerNicks returns a list of 'nick (shortID)' for all peers currently
// subscribed to the room's topic (including ourselves).
func (r *Room) PeerNicks() []string {
	peers := r.topic.ListPeers()
	out := make([]string, 0, len(peers)+1)
	out = append(out, fmt.Sprintf("%s (%s) (you)", r.nick, truncateID(r.selfID.String())))

	r.nicksMu.RLock()
	defer r.nicksMu.RUnlock()

	for _, p := range peers {
		id := p.String()
		shortID := truncateID(id)
		if nick, ok := r.peerNicks[id]; ok {
			out = append(out, fmt.Sprintf("%s (%s)", nick, shortID))
		} else {
			out = append(out, fmt.Sprintf("(%s)", shortID))
		}
	}
	return out
}

func truncateID(id string) string {
	if len(id) > 12 {
		return id[:12] + "…"
	}
	return id
}

// RoomName returns the human-visible room name.
func (r *Room) RoomName() string { return r.roomName }

// Nick returns our current nickname.
func (r *Room) Nick() string { return r.nick }

// SelfID returns our PeerID string.
func (r *Room) SelfID() string { return r.selfID.String() }

// Close sends a leave announcement and unsubscribes from the topic.
func (r *Room) Close() {
	r.once.Do(func() {
		_ = r.Publish(NewLeave(r.selfID.String(), r.nick))
		r.sub.Cancel()
		_ = r.topic.Close()
		close(r.msgs)
		log.Infof("left room %q", r.roomName)
	})
}

// readLoop drains the GossipSub subscription and pushes decoded messages.
func (r *Room) readLoop(ctx context.Context) {
	for {
		rawMsg, err := r.sub.Next(ctx)
		if err != nil {
			return
		}

		if rawMsg.GetFrom() == r.selfID {
			continue
		}

		msg, err := Decode(rawMsg.Data)
		if err != nil {
			log.Debugf("malformed message from %s: %v", rawMsg.GetFrom(), err)
			continue
		}

		// Update nickname map
		r.nicksMu.Lock()
		oldNick, exists := r.peerNicks[msg.SenderID]
		isNew := !exists || (msg.Type == MsgJoin && oldNick != msg.Nick)
		
		if msg.Type == MsgLeave {
			delete(r.peerNicks, msg.SenderID)
		} else if msg.Nick != "" {
			r.peerNicks[msg.SenderID] = msg.Nick
		}
		r.nicksMu.Unlock()

		// For MsgJoin, only push to UI if it's actually new/changed.
		if msg.Type == MsgJoin && !isNew {
			continue
		}

		select {
		case r.msgs <- msg:
		case <-ctx.Done():
			return
		default:
			log.Warn("message buffer full — dropping message")
		}
	}
}

func topicForRoom(name string) string {
	return "/jala/room/v1/" + name
}
