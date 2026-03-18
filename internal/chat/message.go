// Package chat defines the message schema and room management for jala.
package chat

import (
	"encoding/json"
	"fmt"
	"time"
)

// MsgType distinguishes the kinds of events that flow through a room.
type MsgType string

const (
	// MsgChat is a regular user-typed chat message.
	MsgChat MsgType = "chat"

	// MsgJoin is broadcast when a peer enters the room.
	MsgJoin MsgType = "join"

	// MsgLeave is broadcast when a peer exits the room.
	MsgLeave MsgType = "leave"
)

// Message is the wire format for every event in a chat room.
// It is serialised as JSON before being handed to GossipSub.
type Message struct {
	Type      MsgType   `json:"type"`
	SenderID  string    `json:"sender_id"`   // full PeerID string
	Nick      string    `json:"nick"`         // human-readable nickname
	Body      string    `json:"body"`         // text for MsgChat; empty for join/leave
	Timestamp time.Time `json:"ts"`
}

// NewChat constructs a chat message ready to be published.
func NewChat(senderID, nick, body string) Message {
	return Message{
		Type:      MsgChat,
		SenderID:  senderID,
		Nick:      nick,
		Body:      body,
		Timestamp: time.Now().UTC(),
	}
}

// NewJoin constructs a join announcement.
func NewJoin(senderID, nick string) Message {
	return Message{
		Type:      MsgJoin,
		SenderID:  senderID,
		Nick:      nick,
		Timestamp: time.Now().UTC(),
	}
}

// NewLeave constructs a leave announcement.
func NewLeave(senderID, nick string) Message {
	return Message{
		Type:      MsgLeave,
		SenderID:  senderID,
		Nick:      nick,
		Timestamp: time.Now().UTC(),
	}
}

// Encode serialises m to JSON bytes for GossipSub.
func (m Message) Encode() ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("message: encode: %w", err)
	}
	return data, nil
}

// Decode deserialises a JSON-encoded Message.
func Decode(data []byte) (Message, error) {
	var m Message
	if err := json.Unmarshal(data, &m); err != nil {
		return Message{}, fmt.Errorf("message: decode: %w", err)
	}
	return m, nil
}

// DisplayLine returns a formatted string suitable for the chat view.
//
//	[15:04] alice: hello world
//	--- bob joined ---
//	--- carol left ---
func (m Message) DisplayLine() string {
	ts := m.Timestamp.Local().Format("15:04")
	switch m.Type {
	case MsgChat:
		return fmt.Sprintf("[%s] %s: %s", ts, m.Nick, m.Body)
	case MsgJoin:
		return fmt.Sprintf("--- %s joined ---", m.Nick)
	case MsgLeave:
		return fmt.Sprintf("--- %s left ---", m.Nick)
	default:
		return fmt.Sprintf("[%s] <%s> %s", ts, m.Type, m.Body)
	}
}
