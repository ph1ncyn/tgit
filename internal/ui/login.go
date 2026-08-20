package ui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tgit/internal/ghauth"
	"tgit/internal/openurl"
)

// loginResultMsg приходит как из явного входа, так и из тихой проверки
// ранее сохранённого токена при старте — оба случая обрабатывает App.Update.
type loginResultMsg struct {
	user *ghauth.User
	err  error
}

func validateCmd(token string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		u, err := ghauth.ValidateToken(ctx, token)
		return loginResultMsg{user: u, err: err}
	}
}

type loginModel struct {
	input    textinput.Model
	spinner  spinner.Model
	checking bool
	errMsg   string
	width    int
}

func newLoginModel() loginModel {
	ti := textinput.New()
	ti.Placeholder = "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.CharLimit = 255
	ti.Width = 50
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = titleStyle

	return loginModel{input: ti, spinner: sp}
}

func (m loginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m loginModel) tokenValue() string {
	return strings.TrimSpace(m.input.Value())
}

func (m loginModel) Update(msg tea.Msg) (loginModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case spinner.TickMsg:
		if !m.checking {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlO:
			_ = openurl.Open(ghauth.TokenCreateURL)
			return m, nil
		case tea.KeyEnter:
			if m.checking {
				return m, nil
			}
			token := m.tokenValue()
			if token == "" {
				m.errMsg = "вставьте токен перед входом"
				return m, nil
			}
			m.checking = true
			m.errMsg = ""
			return m, tea.Batch(validateCmd(token), m.spinner.Tick)
		}
	}

	if m.checking {
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m loginModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("tgit — вход через GitHub") + "\n\n")
	b.WriteString("Нужен Personal Access Token с правом " + lipgloss.NewStyle().Bold(true).Render("repo") + ".\n")
	b.WriteString("Создать токен: " + linkStyle.Render(Hyperlink(ghauth.TokenCreateURL, ghauth.TokenCreateURL)) + "\n")
	b.WriteString(helpStyle.Render("ctrl+o — открыть эту ссылку в браузере автоматически") + "\n\n")

	b.WriteString("Токен: " + m.input.View() + "\n\n")

	if m.checking {
		b.WriteString(m.spinner.View() + " проверяю токен на GitHub...\n")
	}
	if m.errMsg != "" {
		b.WriteString(errorStyle.Render("✗ "+m.errMsg) + "\n")
	}

	b.WriteString("\n" + helpStyle.Render("enter — войти  •  esc — пропустить и работать локально  •  ctrl+c — выйти"))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
