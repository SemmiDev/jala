package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/semmidev/jala/internal/chat"
)

// renderedMsg is a pre-formatted message line stored in the view buffer.
type renderedMsg struct {
	raw      chat.Message
	rendered string // final lipgloss-styled string
}

// ChatView manages the scrolling message history panel.
type ChatView struct {
	messages []renderedMsg
	selfID   string // our PeerID — used to highlight our own messages
	width    int
	height   int
}

// NewChatView creates an empty chat view.
func NewChatView(selfID string, width, height int) *ChatView {
	return &ChatView{
		selfID: selfID,
		width:  width,
		height: height,
	}
}

// Push adds a message to the view buffer.
func (v *ChatView) Push(msg chat.Message) {
	v.messages = append(v.messages, renderedMsg{
		raw:      msg,
		rendered: v.renderMsg(msg),
	})
}

// Resize updates the panel dimensions and re-renders all messages.
func (v *ChatView) Resize(width, height int) {
	v.width = width
	v.height = height
	for i, m := range v.messages {
		v.messages[i].rendered = v.renderMsg(m.raw)
	}
}

// View returns the visible portion of the message history, filling
// exactly height lines (padding with empty lines at the top if needed).
func (v *ChatView) View() string {
	lines := make([]string, len(v.messages))
	for i, m := range v.messages {
		lines[i] = m.rendered
	}

	// Show only the last `height` lines (scroll to bottom).
	if len(lines) > v.height {
		lines = lines[len(lines)-v.height:]
	}

	// Pad top with empty lines so messages are anchored to the bottom.
	for len(lines) < v.height {
		lines = append([]string{""}, lines...)
	}

	return strings.Join(lines, "\n")
}

// renderMsg converts a Message into a styled, word-wrapped string.
func (v *ChatView) renderMsg(msg chat.Message) string {
	ts := msg.Timestamp.Local().Format("15:04")

	switch msg.Type {
	case chat.MsgJoin:
		return styleMsgSystem.Render(
			fmt.Sprintf("  ─── %s joined ───", msg.Nick))

	case chat.MsgLeave:
		return styleMsgSystem.Render(
			fmt.Sprintf("  ─── %s left ───", msg.Nick))

	case chat.MsgChat:
		timeStr := styleMsgTime.Render(ts)

		var nickStr string
		if msg.SenderID == v.selfID {
			nickStr = styleMsgNickSelf.Render(msg.Nick)
		} else {
			nickStr = styleMsgNickOther.Render(msg.Nick)
		}

		// Word-wrap the body to fit within the panel width.
		// Reserve space for "[15:04] nick: " prefix.
		prefixLen := len(ts) + 3 + len(msg.Nick) + 2 // "[ts] nick: "
		bodyWidth := v.width - prefixLen - 2
		if bodyWidth < 20 {
			bodyWidth = 20
		}
		body := wrapText(msg.Body, bodyWidth)
		// Indent continuation lines to align under the body start.
		indent := strings.Repeat(" ", prefixLen)
		lines := strings.Split(body, "\n")
		for i := 1; i < len(lines); i++ {
			lines[i] = indent + lines[i]
		}
		bodyStr := styleMsgBody.Render(strings.Join(lines, "\n"))

		return fmt.Sprintf("%s %s: %s",
			timeStr,
			nickStr,
			bodyStr,
		)

	default:
		return styleMsgSystem.Render(
			fmt.Sprintf("[%s] <%s> %s", ts, msg.Type, msg.Body))
	}
}

// PeerListView renders the right-side peer list panel.
func PeerListView(peers []string, width int) string {
	var sb strings.Builder

	title := stylePeerCount.Render(fmt.Sprintf("peers (%d)", len(peers)))
	sb.WriteString(lipgloss.PlaceHorizontal(width-2, lipgloss.Center, title))
	sb.WriteString("\n")

	for _, p := range peers {
		// Truncate long peer entries to fit the panel
		if len(p) > width-4 {
			p = p[:width-7] + "…"
		}
		sb.WriteString(stylePeerItem.Render("• "+p) + "\n")
	}

	return stylePeerList.Width(width).Render(sb.String())
}

// StatusBarView renders the bottom status bar.
func StatusBarView(room, nick, peerID string, width int) string {
	roomPart := styleStatusKey.Render("#"+room) + styleStatusBar.Render("  ")
	nickPart := styleStatusBar.Render("nick: ") + styleStatusKey.Render(nick)
	idPart := styleStatusBar.Render("  id: " + truncatePeerID(peerID, 12))
	timePart := styleStatusBar.Render(time.Now().Format("15:04"))

	left := roomPart + nickPart + idPart
	right := timePart

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return styleStatusBar.Width(width).Render(
		left + strings.Repeat(" ", gap) + right)
}

// HelpView renders a one-line keyboard shortcut hint.
func HelpView(width int) string {
	hints := []string{
		styleHelpKey.Render("Enter") + styleHelp.Render(" send"),
		styleHelpKey.Render("Esc") + styleHelp.Render(" quit"),
		styleHelpKey.Render("Tab") + styleHelp.Render(" peers"),
		styleHelpKey.Render("Ctrl+L") + styleHelp.Render(" clear"),
	}
	return styleHelp.Width(width).Render(strings.Join(hints, "  "))
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func truncatePeerID(id string, n int) string {
	r := []rune(id)
	if len(r) <= n {
		return id
	}
	return string(r[:n]) + "…"
}

// wrapText wraps s to at most width characters per line, breaking on spaces.
// This is a simple greedy algorithm — good enough for chat messages.
func wrapText(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}

	var sb strings.Builder
	words := strings.Fields(s)
	lineLen := 0

	for i, w := range words {
		wLen := len(w)
		if lineLen == 0 {
			sb.WriteString(w)
			lineLen = wLen
		} else if lineLen+1+wLen <= width {
			sb.WriteByte(' ')
			sb.WriteString(w)
			lineLen += 1 + wLen
		} else {
			sb.WriteByte('\n')
			sb.WriteString(w)
			lineLen = wLen
		}
		_ = i
	}
	return sb.String()
}
