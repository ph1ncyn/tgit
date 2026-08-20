package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tgit/internal/config"
	"tgit/internal/gitrepo"
)

type noRepoMode int

const (
	noRepoModeList noRepoMode = iota
	noRepoModeClone
)

// recentLoadedMsg — список недавних репозиториев прочитан с диска.
type recentLoadedMsg struct {
	entries []config.RecentEntry
}

// repoOpenedMsg — попытка открыть репозиторий (из недавних или клонированием)
// завершена; обрабатывается на уровне App, которая при успехе переключает
// приложение на главный экран с новым Repo.
type repoOpenedMsg struct {
	repo *gitrepo.Repo
	err  error
}

func loadRecentCmd() tea.Cmd {
	return func() tea.Msg {
		entries, _ := config.LoadRecent()
		return recentLoadedMsg{entries: entries}
	}
}

func openRecentCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if err := os.Chdir(path); err != nil {
			return repoOpenedMsg{err: err}
		}
		repo, err := gitrepo.Open(path)
		if err != nil {
			return repoOpenedMsg{err: err}
		}
		_ = config.AddRecent(repo.Root)
		return repoOpenedMsg{repo: repo}
	}
}

func cloneCmd(url, dir string) tea.Cmd {
	return func() tea.Msg {
		repo, err := gitrepo.Clone(url, dir)
		if err != nil {
			return repoOpenedMsg{err: err}
		}
		_ = config.AddRecent(repo.Root)
		return repoOpenedMsg{repo: repo}
	}
}

// noRepoModel — экран, который показывается вместо главного, когда в
// текущей папке нет git-репозитория: предлагает открыть один из недавних
// проектов tgit или склонировать репозиторий прямо в эту папку.
type noRepoModel struct {
	dir     string
	entries []config.RecentEntry
	cursor  int

	mode noRepoMode
	url  textinput.Model

	busy      bool
	busyLabel string
	spinner   spinner.Model
	err       string

	width int
}

func newNoRepoModel(dir string) noRepoModel {
	ti := textinput.New()
	ti.Placeholder = "https://github.com/user/repo.git"
	ti.Width = 60

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = titleStyle

	return noRepoModel{dir: dir, url: ti, spinner: sp}
}

func (m noRepoModel) Init() tea.Cmd {
	return loadRecentCmd()
}

// IsInputActive сообщает, активно ли сейчас поле ввода URL — в этом
// состоянии однобуквенные горячие клавиши (например 'g') должны попадать в
// поле, а не перехватываться глобально.
func (m noRepoModel) IsInputActive() bool {
	return m.mode == noRepoModeClone
}

func (m noRepoModel) Update(msg tea.Msg) (noRepoModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case spinner.TickMsg:
		if !m.busy {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case recentLoadedMsg:
		m.entries = msg.entries
		m.cursor = clamp(m.cursor, 0, maxInt(len(m.entries)-1, 0))
		return m, nil

	case repoOpenedMsg:
		// Успех переключает состояние приложения в App.Update; здесь только
		// снимаем busy и показываем ошибку, если она была.
		m.busy = false
		if msg.err != nil {
			m.err = msg.err.Error()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m noRepoModel) handleKey(msg tea.KeyMsg) (noRepoModel, tea.Cmd) {
	if m.busy {
		return m, nil
	}

	if m.mode == noRepoModeClone {
		switch msg.Type {
		case tea.KeyEsc:
			m.mode = noRepoModeList
			m.url.Blur()
			m.err = ""
			return m, nil
		case tea.KeyEnter:
			url := strings.TrimSpace(m.url.Value())
			if url == "" {
				m.err = "введите URL репозитория"
				return m, nil
			}
			m.err = ""
			m.busy, m.busyLabel = true, "клонирую..."
			return m, tea.Batch(cloneCmd(url, m.dir), m.spinner.Tick)
		}
		var cmd tea.Cmd
		m.url, cmd = m.url.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "up", "k":
		if len(m.entries) > 0 {
			m.cursor = clamp(m.cursor-1, 0, len(m.entries)-1)
		}
	case "down", "j":
		if len(m.entries) > 0 {
			m.cursor = clamp(m.cursor+1, 0, len(m.entries)-1)
		}
	case "enter":
		if m.cursor < len(m.entries) {
			m.err = ""
			m.busy, m.busyLabel = true, "открываю проект..."
			return m, tea.Batch(openRecentCmd(m.entries[m.cursor].Path), m.spinner.Tick)
		}
	case "c":
		m.mode = noRepoModeClone
		m.err = ""
		m.url.SetValue("")
		m.url.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m noRepoModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("tgit") + "\n\n")
	b.WriteString(errorStyle.Render("В папке "+m.dir+" не найден git-репозиторий.") + "\n\n")

	if m.mode == noRepoModeClone {
		b.WriteString("Клонировать сюда: " + m.url.View() + "\n\n")
		if m.busy {
			b.WriteString(m.spinner.View() + " " + helpStyle.Render(m.busyLabel) + "\n\n")
		}
		if m.err != "" {
			b.WriteString(errorStyle.Render("✗ "+m.err) + "\n\n")
		}
		b.WriteString(helpStyle.Render("enter — клонировать  •  esc — назад  •  ctrl+c — выйти"))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if len(m.entries) == 0 {
		b.WriteString(helpStyle.Render("недавних проектов tgit пока нет") + "\n\n")
	} else {
		b.WriteString(helpStyle.Render("Недавние проекты:") + "\n")
		for i, e := range m.entries {
			prefix := "  "
			style := lipgloss.NewStyle()
			if i == m.cursor {
				prefix, style = "> ", style.Reverse(true)
			}
			b.WriteString(style.Render(prefix+e.Path) + "\n")
		}
		b.WriteString("\n")
	}

	if m.busy {
		b.WriteString(m.spinner.View() + " " + helpStyle.Render(m.busyLabel) + "\n\n")
	}
	if m.err != "" {
		b.WriteString(errorStyle.Render("✗ "+m.err) + "\n\n")
	}

	help := "c — клонировать репозиторий в эту папку  •  ctrl+c — выйти"
	if len(m.entries) > 0 {
		help = "↑/↓ — выбрать  •  enter — открыть проект  •  " + help
	}
	b.WriteString(helpStyle.Render(help))
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
