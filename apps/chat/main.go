package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func main() {
	log.SetFlags(0)

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0644)

	if err != nil {
		log.Fatal(err)
	}

	lipgloss.SetColorProfile(termenv.NewOutput(tty).EnvColorProfile())

	opts := []tea.ProgramOption{
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	}

	p := tea.NewProgram(initialModel(), opts...)

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type (
	chatMsg   string
	closedMsg struct{}
	errMsg    error
)

func readInput(inbox chan string) tea.Cmd {
	return func() tea.Msg {
		input := bufio.NewScanner(os.Stdin)

		for input.Scan() {
			inbox <- input.Text()
		}

		if err := input.Err(); err != nil {
			return []tea.Msg{errMsg(err), closedMsg{}}
		}

		return closedMsg{}
	}
}

func takeInput(inbox chan string) tea.Cmd {
	return func() tea.Msg {
		return chatMsg(<-inbox)
	}
}

var (
	senderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	otherStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

type model struct {
	viewport viewport.Model
	textarea textarea.Model
	messages []string
	inbox    chan string
	err      error
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = lipgloss.ThickBorder().Left + " "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(30, 5)
	vp.SetContent("Type a message and press Enter to send.")

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return model{
		viewport: vp,
		textarea: ta,
		messages: []string{},
		inbox:    make(chan string),
		err:      nil,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		readInput(m.inbox),
		takeInput(m.inbox),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			message := m.textarea.Value()

			os.Stdout.WriteString(fmt.Sprintf("%v\n", message))

			m.messages = append(m.messages, senderStyle.Render("You:  ")+message)
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.textarea.Reset()
			m.viewport.GotoBottom()
		}

	case chatMsg:
		m.messages = append(m.messages, otherStyle.Render("They: ")+string(msg))
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
		cmds = append(cmds, takeInput(m.inbox))

	case closedMsg:
		return m, tea.Quit

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n",
		m.viewport.View(),
		m.textarea.View(),
	)
}
