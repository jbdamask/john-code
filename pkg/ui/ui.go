package ui

import (
	"fmt"
    "io/ioutil"
    "path/filepath"
	"strings"
    "time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
    "golang.design/x/clipboard"
)

type UI struct{}

func New() *UI {
	return &UI{}
}

func (u *UI) Print(msg string) {
	fmt.Println(msg)
}

// Input Handling

type inputModel struct {
	textInput textinput.Model
	err       error
	output    string
	canceled  bool
}

func initialInputModel(prompt string) inputModel {
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 80
    ti.Prompt = prompt

	return inputModel{
		textInput: ti,
		err:       nil,
	}
}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.output = m.textInput.Value()
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
            m.canceled = true
			return m, tea.Quit
        case tea.KeyCtrlV:
            // Check for image data in clipboard
            err := clipboard.Init()
            if err == nil {
                imageBytes := clipboard.Read(clipboard.FmtImage)
                if len(imageBytes) > 0 {
                    // Save to temp file
                    tmpDir := "/tmp" // Cross platform consideration needed? For MVP /tmp is ok
                    filename := fmt.Sprintf("john_clipboard_%d.png", time.Now().UnixNano())
                    path := filepath.Join(tmpDir, filename)
                    
                    if err := ioutil.WriteFile(path, imageBytes, 0644); err == nil {
                        m.textInput.SetValue(m.textInput.Value() + fmt.Sprintf(" [Image: %s] ", path))
                        // Position cursor at end
                        m.textInput.SetCursor(len(m.textInput.Value()))
                    }
                }
            }
		}
	case error:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	return fmt.Sprintf(
		"%s\n",
		m.textInput.View(),
	)
}

func (u *UI) Prompt(prompt string) string {
	p := tea.NewProgram(initialInputModel(prompt))
	m, err := p.Run()
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		return ""
	}

	if mModel, ok := m.(inputModel); ok {
        if mModel.canceled {
            return "exit"
        }
		return strings.TrimSpace(mModel.output)
	}
	return ""
}

// Stream Handling

type streamModel struct {
	sub      <-chan string
	content  string
	showing  bool
	finished bool
}

type tokenMsg string
type finishMsg struct{}

func waitForToken(sub <-chan string) tea.Cmd {
	return func() tea.Msg {
		token, ok := <-sub
		if !ok {
			return finishMsg{}
		}
		return tokenMsg(token)
	}
}

func (m streamModel) Init() tea.Cmd {
	return waitForToken(m.sub)
}

func (m streamModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+o" {
			m.showing = !m.showing
			return m, nil
		}
        if msg.Type == tea.KeyCtrlC {
            return m, tea.Quit
        }
	case tokenMsg:
		m.content += string(msg)
		return m, waitForToken(m.sub)
	case finishMsg:
		m.finished = true
        // Ensure we show the content at the end
        m.showing = true
		return m, tea.Quit
	}
	return m, nil
}

func (m streamModel) View() string {
	if !m.showing {
		return "Thinking... (Press Ctrl+O to show stream)"
	}
	return m.content
}

func (u *UI) DisplayStream(outputChan <-chan string) {
	m := streamModel{
		sub:     outputChan,
		showing: true, // Default to showing
	}
	p := tea.NewProgram(m)
	_, err := p.Run()
	if err != nil {
		fmt.Printf("Error in stream display: %v\n", err)
	}
}
