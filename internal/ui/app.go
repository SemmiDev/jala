package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/semmidev/jala/internal/chat"
)

// ── Messages (Bubble Tea Msg types) ──────────────────────────────────────────

// incomingMsg carries a chat.Message from the network into the UI event loop.
type incomingMsg chat.Message

// peerListMsg carries an updated peer list snapshot.
type peerListMsg []string

// errMsg carries a non-fatal error to display in the status bar.
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// roomClosedMsg is sent when the room's message channel is closed
// (the node shut down). Handled in Update to cleanly quit the TUI.
type roomClosedMsg struct{}

// clearStatusMsg is sent after a short delay to dismiss transient
// error banners from the status bar.
type clearStatusMsg struct{}

// ── Model ─────────────────────────────────────────────────────────────────────

// Model is the Bubble Tea root model. It owns all UI state.
//
// Following Elm Architecture:
//   - Model  holds state (immutable snapshot per update cycle)
//   - Update transforms state in response to Msg events
//   - View   renders state to a string; called after every Update
//
// Goroutine safety: Bubble Tea guarantees that Update is called
// serially on the same goroutine, so no locks are needed on Model fields.
type Model struct {
	// Subsystem references (read-only after Init)
	room   *chat.Room
	selfID string

	// UI components
	input    textinput.Model
	chatView *ChatView

	// State
	peers     []string
	showPeers bool
	statusMsg string // transient status line (errors, etc.)
	quitting  bool

	// Terminal dimensions
	width  int
	height int
}

// New creates a Model ready for tea.NewProgram.
func New(room *chat.Room) *Model {
	ti := textinput.New()
	ti.Placeholder = "type a message… (Esc to quit)"
	ti.Focus()
	ti.CharLimit = 1000

	m := &Model{
		room:   room,
		selfID: room.SelfID(),
		input:  ti,
		// chatView will be sized on WindowSizeMsg
		chatView: NewChatView(room.SelfID(), 80, 20),
	}
	return m
}

// ── Init ──────────────────────────────────────────────────────────────────────

// Init starts the Bubble Tea program and kicks off the goroutine that
// bridges the chat.Room.Messages channel into the Bubble Tea event loop.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.listenForMessages(),
		m.refreshPeerList(),
		m.tick(),
	)
}

// listenForMessages returns a Cmd that waits for the next message from
// the room and delivers it as an incomingMsg.
//
// Bubble Tea Cmds are run in separate goroutines and deliver a single Msg.
// By returning a new listenForMessages() Cmd from Update each time we
// receive a message, we create a continuous listener without blocking.
func (m *Model) listenForMessages() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.room.Messages
		if !ok {
			// Channel closed — the room was shut down externally.
			// Return a typed Msg so Update can dispatch tea.Quit.
			// (tea.Quit() itself is a Cmd, not a Msg — returning it
			// directly here would be silently ignored by Bubble Tea.)
			return roomClosedMsg{}
		}
		return incomingMsg(msg)
	}
}

// refreshPeerList returns a Cmd that snapshots the current peer list.
func (m *Model) refreshPeerList() tea.Cmd {
	return func() tea.Msg {
		return peerListMsg(m.room.PeerNicks())
	}
}

// tickMsg is sent periodically to refresh background state.
type tickMsg struct{}

func (m *Model) tick() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// ── Terminal resize ───────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.chatView.Resize(m.chatPanelWidth(), m.chatPanelHeight())

	// ── Keyboard ──────────────────────────────────────────────────────────
	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyEsc, tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			if text := strings.TrimSpace(m.input.Value()); text != "" {
				m.input.Reset()
				cmds = append(cmds, m.sendMessage(text))
			}

		case tea.KeyTab:
			m.showPeers = !m.showPeers
			if m.showPeers {
				cmds = append(cmds, m.refreshPeerList())
			}

		case tea.KeyCtrlL:
			// Clear chat history
			m.chatView = NewChatView(m.selfID, m.chatPanelWidth(), m.chatPanelHeight())

		default:
			// All other keys go to the text input
			var inputCmd tea.Cmd
			m.input, inputCmd = m.input.Update(msg)
			cmds = append(cmds, inputCmd)
		}

	// ── Room closed (network shutdown) ───────────────────────────────────
	case roomClosedMsg:
		return m, tea.Quit

	// ── Incoming network message ──────────────────────────────────────────
	case incomingMsg:
		m.chatView.Push(chat.Message(msg))
		// Re-arm the listener so we get the next message
		cmds = append(cmds, m.listenForMessages())
		// Refresh peer list on join/leave events
		if msg.Type == chat.MsgJoin || msg.Type == chat.MsgLeave {
			cmds = append(cmds, m.refreshPeerList())
		}

	// ── Peer list update ──────────────────────────────────────────────────
	case peerListMsg:
		m.peers = []string(msg)

	case tickMsg:
		cmds = append(cmds, m.refreshPeerList(), m.tick())

	// ── Error ─────────────────────────────────────────────────────────────
	case errMsg:
		m.statusMsg = "⚠ " + msg.Error()
		// Auto-clear the banner after 4 s so it doesn't permanently
		// obscure the real status bar.
		cmds = append(cmds, tea.Tick(4*time.Second, func(time.Time) tea.Msg {
			return clearStatusMsg{}
		}))

	case clearStatusMsg:
		m.statusMsg = ""
	}

	return m, tea.Batch(cmds...)
}

// sendMessage publishes text to the room and echoes it locally.
func (m *Model) sendMessage(text string) tea.Cmd {
	return func() tea.Msg {
		if err := m.room.Send(text); err != nil {
			return errMsg{err}
		}
		// Echo our own message immediately (GossipSub won't deliver it back)
		selfMsg := chat.NewChat(m.selfID, m.room.Nick(), text)
		return incomingMsg(selfMsg)
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m *Model) View() string {
	if m.quitting {
		return styleTitle.Render("jala") + "\n" +
			styleMsgSystem.Render("goodbye!") + "\n"
	}

	if m.width == 0 {
		return "initialising…\n"
	}

	// Layout:
	//
	//   ┌────────────────────────────┬──────────────┐
	//   │  title bar                 │              │
	//   ├────────────────────────────│  peer list   │
	//   │                            │  (optional)  │
	//   │  chat messages             │              │
	//   │                            │              │
	//   ├────────────────────────────┴──────────────┤
	//   │  input box                                 │
	//   ├────────────────────────────────────────────┤
	//   │  status bar                                │
	//   ├────────────────────────────────────────────┤
	//   │  help bar                                  │
	//   └────────────────────────────────────────────┘

	var sections []string

	// Title
	title := styleTitle.Render("jala") +
		styleMsgSystem.Render(fmt.Sprintf(" #%s", m.room.RoomName()))
	sections = append(sections, title)

	// Main content area: chat + optional peer panel
	chatPanel := m.chatView.View()

	if m.showPeers {
		peerW := m.peerPanelWidth()
		chatW := m.chatPanelWidth()
		peerPanel := PeerListView(m.peers, peerW)

		// Side-by-side join
		content := lipgloss.JoinHorizontal(lipgloss.Top,
			styleBorder.Width(chatW).Height(m.chatPanelHeight()).Render(chatPanel),
			peerPanel,
		)
		sections = append(sections, content)
	} else {
		sections = append(sections,
			styleBorder.Width(m.width-2).Height(m.chatPanelHeight()).Render(chatPanel))
	}

	// Input row
	prompt := styleInputPrompt.Render("> ")
	inputLine := styleInput.Width(m.width - 2).Render(
		prompt + m.input.View())
	sections = append(sections, inputLine)

	// Status bar
	status := StatusBarView(m.room.RoomName(), m.room.Nick(), m.selfID, m.width)
	if m.statusMsg != "" {
		status = styleStatusBar.Width(m.width).Render(m.statusMsg)
	}
	sections = append(sections, status)

	// Help bar
	sections = append(sections, HelpView(m.width))

	return strings.Join(sections, "\n")
}

// ── Layout helpers ────────────────────────────────────────────────────────────

const (
	peerPanelDefaultWidth = 22
	fixedRowCount         = 6 // title + border overhead + input + status + help + 1 spare
)

func (m *Model) chatPanelHeight() int {
	h := m.height - fixedRowCount
	if h < 5 {
		return 5
	}
	return h
}

func (m *Model) chatPanelWidth() int {
	if m.showPeers {
		w := m.width - peerPanelDefaultWidth - 4
		if w < 30 {
			return 30
		}
		return w
	}
	return m.width - 4
}

func (m *Model) peerPanelWidth() int {
	return peerPanelDefaultWidth
}

// ── Program constructor ───────────────────────────────────────────────────────

// Run starts the Bubble Tea program and blocks until the user quits.
// ctx is not directly used by Bubble Tea but signals when the node
// has shut down externally (e.g. SIGINT).
func Run(ctx context.Context, room *chat.Room) error {
	model := New(room)
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // full-screen mode, restores terminal on exit
		tea.WithMouseCellMotion(), // enables mouse scroll in future
	)

	// If context is cancelled (e.g. SIGINT before TUI starts), quit the program.
	go func() {
		<-ctx.Done()
		p.Quit()
	}()

	_, err := p.Run()
	return err
}
