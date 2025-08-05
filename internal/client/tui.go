package client

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"encoding/json"
	"go-chat/pkg/chat"
)

type Model struct {
	isEnteringUsername bool
	messages           []string
	input              textinput.Model
	username           string
	ws                 *WSClient
	connected          bool
	msgChan            chan tea.Msg
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Type your message here"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	ch := make(chan tea.Msg, 10)
	ws, err := NewWSClient(ch)
	if err != nil {
		panic(err)
	}

	return Model{
		input:              ti,
		username:           "user",
		ws:                 ws,
		connected:          true,
		isEnteringUsername: true,
		msgChan:            ch,
	}
}

func (m Model) Init() tea.Cmd {
	textinput.Blink()
	return func() tea.Msg {
		return <-m.msgChan
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			text := m.input.Value()

			if m.isEnteringUsername {
				m.username = text
				m.isEnteringUsername = false
				m.input.Placeholder = "Type your message here"
				m.input.Reset()

				ws, err := NewWSClient(m.msgChan)

				if err != nil {
					panic(err)
				}

				ws.Start()
				m.ws = ws
				m.connected = true

				return m, nil
			}

			m.input.Reset()

			_ = m.ws.Send(chat.Message{
				Username: m.username,
				Text:     text,
			})

			return m, nil
		default:
			m.input, cmd = m.input.Update(msg)
		}

	case messageReceivedMsg:
		var incoming chat.Message
		if err := json.Unmarshal([]byte(msg), &incoming); err == nil {
			m.messages = append(m.messages, fmt.Sprintf("[%s]: %s", incoming.Username, incoming.Text))
		}
		return m, func() tea.Msg {
			return <-m.msgChan
		}
	}

	return m, cmd
}

func (m Model) View() string {
	if m.isEnteringUsername {
		return fmt.Sprintf("Enter your username: %s\n", m.input.View())
	}
	var b strings.Builder

	for _, msg := range m.messages {
		b.WriteString(msg + "\n")
	}

	b.WriteString("\n" + m.input.View())
	b.WriteString("\n[Enter] to send")
	return b.String()
}
