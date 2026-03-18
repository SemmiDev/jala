package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/semmidev/jala/internal/chat"
)

func TestChatViewPushAndView(t *testing.T) {
	v := NewChatView("self123", 80, 10)

	msg := chat.Message{
		Type:      chat.MsgChat,
		SenderID:  "peer456",
		Nick:      "alice",
		Body:      "hello!",
		Timestamp: time.Now(),
	}
	v.Push(msg)

	view := v.View()
	if !strings.Contains(view, "alice") {
		t.Errorf("expected nick in view, got:\n%s", view)
	}
	if !strings.Contains(view, "hello!") {
		t.Errorf("expected body in view, got:\n%s", view)
	}
}

func TestChatViewSelfHighlight(t *testing.T) {
	selfID := "self123"
	v := NewChatView(selfID, 80, 10)

	selfMsg := chat.Message{
		Type: chat.MsgChat, SenderID: selfID,
		Nick: "me", Body: "my message", Timestamp: time.Now(),
	}
	v.Push(selfMsg)
	view := v.View()
	// Both messages should appear
	if !strings.Contains(view, "my message") {
		t.Errorf("self message not rendered:\n%s", view)
	}
}

func TestChatViewHeight(t *testing.T) {
	v := NewChatView("self", 80, 5)

	// Push more messages than the view height
	for i := 0; i < 20; i++ {
		v.Push(chat.Message{
			Type: chat.MsgChat, Nick: "user",
			Body: "line", Timestamp: time.Now(),
		})
	}

	view := v.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}
}

func TestChatViewResize(t *testing.T) {
	v := NewChatView("self", 80, 10)
	v.Push(chat.Message{
		Type: chat.MsgChat, Nick: "a", Body: "hi", Timestamp: time.Now(),
	})

	v.Resize(120, 20)
	if v.width != 120 || v.height != 20 {
		t.Errorf("resize failed: got %dx%d", v.width, v.height)
	}
}

func TestWrapText(t *testing.T) {
	cases := []struct {
		input     string
		width     int
		wantLines int
	}{
		{"short text", 80, 1},
		{"a b c d e f g", 5, 4},
		{"", 80, 1},
		{"oneword", 3, 1}, // single word always on one line
	}

	for _, tc := range cases {
		result := wrapText(tc.input, tc.width)
		lines := strings.Split(result, "\n")
		if len(lines) != tc.wantLines {
			t.Errorf("wrapText(%q, %d): got %d lines, want %d\n  result: %q",
				tc.input, tc.width, len(lines), tc.wantLines, result)
		}
	}
}

func TestTruncatePeerID(t *testing.T) {
	long := "12D3KooWNzeutNDuGHZSDgqpT9NLt8jsGZLiqALBKi14ABdFVrHG"
	got := truncatePeerID(long, 12)
	if len([]rune(got)) > 13 { // 12 + "…"
		t.Errorf("truncation too long: %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected trailing ellipsis: %q", got)
	}

	short := "abc"
	if truncatePeerID(short, 12) != short {
		t.Errorf("short string should not be truncated")
	}
}
