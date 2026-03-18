package chat_test

import (
	"strings"
	"testing"
	"time"

	"github.com/semmidev/jala/internal/chat"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	original := chat.NewChat("peer123", "alice", "hello world")

	data, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	got, err := chat.Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if got.Type != chat.MsgChat {
		t.Errorf("Type: got %q, want %q", got.Type, chat.MsgChat)
	}
	if got.SenderID != "peer123" {
		t.Errorf("SenderID: got %q", got.SenderID)
	}
	if got.Nick != "alice" {
		t.Errorf("Nick: got %q", got.Nick)
	}
	if got.Body != "hello world" {
		t.Errorf("Body: got %q", got.Body)
	}
}

func TestDecodeInvalid(t *testing.T) {
	_, err := chat.Decode([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDisplayLineChat(t *testing.T) {
	msg := chat.Message{
		Type:      chat.MsgChat,
		Nick:      "alice",
		Body:      "hi there",
		Timestamp: time.Date(2025, 1, 1, 15, 4, 0, 0, time.UTC),
	}
	line := msg.DisplayLine()
	if !strings.Contains(line, "alice") {
		t.Errorf("expected nick in display line: %q", line)
	}
	if !strings.Contains(line, "hi there") {
		t.Errorf("expected body in display line: %q", line)
	}
}

func TestDisplayLineJoin(t *testing.T) {
	msg := chat.NewJoin("peer1", "bob")
	line := msg.DisplayLine()
	if !strings.Contains(line, "bob") {
		t.Errorf("expected nick in join line: %q", line)
	}
	if !strings.Contains(line, "joined") {
		t.Errorf("expected 'joined' in join line: %q", line)
	}
}

func TestDisplayLineLeave(t *testing.T) {
	msg := chat.NewLeave("peer2", "carol")
	line := msg.DisplayLine()
	if !strings.Contains(line, "carol") {
		t.Errorf("expected nick in leave line: %q", line)
	}
	if !strings.Contains(line, "left") {
		t.Errorf("expected 'left' in leave line: %q", line)
	}
}

func TestMessageTypes(t *testing.T) {
	cases := []struct {
		msg      chat.Message
		wantType chat.MsgType
	}{
		{chat.NewChat("p1", "a", "hello"), chat.MsgChat},
		{chat.NewJoin("p1", "a"), chat.MsgJoin},
		{chat.NewLeave("p1", "a"), chat.MsgLeave},
	}
	for _, tc := range cases {
		if tc.msg.Type != tc.wantType {
			t.Errorf("got type %q, want %q", tc.msg.Type, tc.wantType)
		}
		if tc.msg.Timestamp.IsZero() {
			t.Error("Timestamp should be set")
		}
	}
}
