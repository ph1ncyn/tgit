package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"tgit/internal/gitrepo"
	"tgit/internal/secret"
)

type screenState int

const (
	stateCheckingToken screenState = iota
	stateLogin
	stateMain
)

// App — корневая модель Bubble Tea, переключающая экран входа и главный экран.
type App struct {
	state screenState
	login loginModel
	main  mainModel

	pendingToken  string
	width, height int
}

// NewApp собирает приложение. Если в системном хранилище уже есть токен,
// приложение тихо проверит его перед показом главного экрана; если токена
// нет — сразу показывает экран входа со ссылкой на создание токена.
func NewApp(repo *gitrepo.Repo) App {
	a := App{
		login: newLoginModel(),
		main:  newMainModel(repo),
	}
	if tok, err := secret.Load(); err == nil && tok != "" {
		a.state = stateCheckingToken
		a.pendingToken = tok
		a.login.checking = true
	} else {
		a.state = stateLogin
	}
	return a
}

func (a App) Init() tea.Cmd {
	cmds := []tea.Cmd{a.main.Init()}
	switch a.state {
	case stateCheckingToken:
		cmds = append(cmds, validateCmd(a.pendingToken), a.login.spinner.Tick)
	case stateLogin:
		cmds = append(cmds, a.login.Init())
	}
	return tea.Batch(cmds...)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		var lcmd, mcmd tea.Cmd
		a.login, lcmd = a.login.Update(msg)
		a.main, mcmd = a.main.Update(msg)
		return a, tea.Batch(lcmd, mcmd)

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return a, tea.Quit
		}
		switch a.state {
		case stateCheckingToken:
			if msg.Type == tea.KeyEsc {
				a.state = stateMain
			}
			return a, nil

		case stateLogin:
			if msg.Type == tea.KeyEsc {
				a.state = stateMain
				return a, nil
			}
			var cmd tea.Cmd
			a.login, cmd = a.login.Update(msg)
			return a, cmd

		case stateMain:
			if msg.String() == "g" {
				a.state = stateLogin
				a.login = newLoginModel()
				return a, a.login.Init()
			}
			var cmd tea.Cmd
			a.main, cmd = a.main.Update(msg)
			return a, cmd
		}
		return a, nil

	case loginResultMsg:
		wasSilentCheck := a.state == stateCheckingToken
		if msg.err != nil {
			a.login.checking = false
			if wasSilentCheck {
				_ = secret.Delete()
				a.login.errMsg = "сохранённый токен недействителен, войдите заново"
				a.state = stateLogin
				return a, nil
			}
			a.login.errMsg = msg.err.Error()
			return a, nil
		}
		if !wasSilentCheck {
			_ = secret.Save(a.login.tokenValue())
		}
		a.main.ghUser = msg.user
		a.state = stateMain
		return a, nil

	default:
		var lcmd, mcmd tea.Cmd
		a.login, lcmd = a.login.Update(msg)
		a.main, mcmd = a.main.Update(msg)
		return a, tea.Batch(lcmd, mcmd)
	}
}

func (a App) View() string {
	switch a.state {
	case stateCheckingToken, stateLogin:
		return a.login.View()
	default:
		return a.main.View()
	}
}
