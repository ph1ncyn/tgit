package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tgit/internal/ghauth"
	"tgit/internal/gitrepo"
)

type repoDataMsg struct {
	branch   string
	branches []string
	files    []gitrepo.FileStatus
	err      error
}

type mainModel struct {
	repo   *gitrepo.Repo
	ghUser *ghauth.User

	branch   string
	branches []string
	files    []gitrepo.FileStatus
	err      string

	focus         int // 0 = ветки, 1 = файлы
	width, height int
}

func newMainModel(repo *gitrepo.Repo) mainModel {
	return mainModel{repo: repo}
}

func (m mainModel) loadCmd() tea.Cmd {
	repo := m.repo
	return func() tea.Msg {
		if repo == nil {
			return repoDataMsg{err: fmt.Errorf("текущий каталог не git-репозиторий")}
		}
		branch, err := repo.CurrentBranch()
		if err != nil {
			return repoDataMsg{err: err}
		}
		branches, err := repo.Branches()
		if err != nil {
			return repoDataMsg{err: err}
		}
		files, err := repo.Status()
		if err != nil {
			return repoDataMsg{err: err}
		}
		return repoDataMsg{branch: branch, branches: branches, files: files}
	}
}

func (m mainModel) Init() tea.Cmd {
	return m.loadCmd()
}

func (m mainModel) Update(msg tea.Msg) (mainModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case repoDataMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
			m.branch, m.branches, m.files = msg.branch, msg.branches, msg.files
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.focus = (m.focus + 1) % 2
		case "r":
			return m, m.loadCmd()
		}
	}
	return m, nil
}

func (m mainModel) View() string {
	width := m.width
	if width <= 0 {
		width = 80
	}

	ghStatus := helpStyle.Render("GitHub: не подключён (g — войти)")
	if m.ghUser != nil {
		label := m.ghUser.Login
		if m.ghUser.Name != "" {
			label = fmt.Sprintf("%s (@%s)", m.ghUser.Name, m.ghUser.Login)
		}
		ghStatus = okStyle.Render("GitHub: " + label)
	}

	branchLabel := m.branch
	if branchLabel == "" {
		branchLabel = "—"
	}
	top := titleStyle.Render("tgit") + "   " + helpStyle.Render("⎇ "+branchLabel) + "   " + ghStatus
	if m.err != "" {
		top += "\n" + errorStyle.Render(m.err)
	}

	panelWidth := (width - 8) / 2
	if panelWidth < 20 {
		panelWidth = 20
	}

	branchesBody := helpStyle.Render("нет данных")
	if len(m.branches) > 0 {
		var lines []string
		for _, b := range m.branches {
			prefix := "  "
			style := lipgloss.NewStyle()
			if b == m.branch {
				prefix = "⎇ "
				style = okStyle
			}
			lines = append(lines, style.Render(prefix+b))
		}
		branchesBody = strings.Join(lines, "\n")
	}
	branchesPanel := panelStyleFor(m.focus == 0).Width(panelWidth).Render(
		titleStyle.Render("Ветки") + "\n" + branchesBody,
	)

	filesBody := okStyle.Render("чисто")
	if len(m.files) > 0 {
		var lines []string
		for _, f := range m.files {
			lines = append(lines, fmt.Sprintf("%c%c %s", f.X, f.Y, f.Path))
		}
		filesBody = strings.Join(lines, "\n")
	}
	filesPanel := panelStyleFor(m.focus == 1).Width(panelWidth).Render(
		titleStyle.Render("Файлы") + "\n" + filesBody,
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, branchesPanel, filesPanel)

	help := helpStyle.Render("tab — переключить панель  •  r — обновить  •  g — GitHub вход  •  ctrl+c — выйти")

	return lipgloss.NewStyle().Padding(1, 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, top, "", body, "", help),
	)
}
