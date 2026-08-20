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
	stateNoRepo
)

// App — корневая модель Bubble Tea, переключающая экран входа, главный экран
// и экран "репозиторий не найден".
type App struct {
	state  screenState
	login  loginModel
	main   mainModel
	noRepo noRepoModel

	pendingToken  string
	width, height int
}

// NewApp собирает приложение. Если в системном хранилище уже есть токен,
// приложение тихо проверит его перед показом главного экрана; если токена
// нет — сразу показывает экран входа со ссылкой на создание токена. Если repo
// не открылся (текущая папка — не git-репозиторий), после экрана входа
// показывается не главный экран, а stateNoRepo.
func NewApp(repo *gitrepo.Repo, cwd string) App {
	a := App{
		login:  newLoginModel(),
		main:   newMainModel(repo),
		noRepo: newNoRepoModel(cwd),
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

// afterAuthState — какой экран показать после того, как экран входа
// (пройденный или пропущенный) закрывается: главный, если репозиторий открыт,
// иначе экран "репозиторий не найден".
func (a App) afterAuthState() screenState {
	if a.main.repo == nil {
		return stateNoRepo
	}
	return stateMain
}

func (a App) Init() tea.Cmd {
	cmds := []tea.Cmd{a.main.Init(), a.noRepo.Init()}
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
		var lcmd, mcmd, ncmd tea.Cmd
		a.login, lcmd = a.login.Update(msg)
		a.main, mcmd = a.main.Update(msg)
		a.noRepo, ncmd = a.noRepo.Update(msg)
		return a, tea.Batch(lcmd, mcmd, ncmd)

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return a, tea.Quit
		}
		switch a.state {
		case stateCheckingToken:
			if msg.Type == tea.KeyEsc {
				a.state = a.afterAuthState()
			}
			return a, nil

		case stateLogin:
			if msg.Type == tea.KeyEsc {
				a.state = a.afterAuthState()
				return a, nil
			}
			var cmd tea.Cmd
			a.login, cmd = a.login.Update(msg)
			return a, cmd

		case stateMain:
			// 'g' переключает на экран входа, но только когда на главном экране
			// нет активного диалога с текстовым полем (коммит, фильтр веток) —
			// иначе буква 'g' должна печататься в это поле, а не перехватываться.
			if msg.String() == "g" && !a.main.IsModal() {
				a.state = stateLogin
				a.login = newLoginModel()
				return a, a.login.Init()
			}
			var cmd tea.Cmd
			a.main, cmd = a.main.Update(msg)
			return a, cmd

		case stateNoRepo:
			// как и на главном экране, 'g' — глобальный переход к логину, но
			// не когда пользователь печатает URL для клонирования.
			if msg.String() == "g" && !a.noRepo.IsInputActive() {
				a.state = stateLogin
				a.login = newLoginModel()
				return a, a.login.Init()
			}
			var cmd tea.Cmd
			a.noRepo, cmd = a.noRepo.Update(msg)
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
		if wasSilentCheck {
			a.main.ghToken = a.pendingToken
		} else {
			a.main.ghToken = a.login.tokenValue()
			_ = secret.Save(a.main.ghToken)
		}
		a.main.ghUser = msg.user
		a.state = a.afterAuthState()
		return a, nil

	// repoOpenedMsg приходит из stateNoRepo (открытие недавнего проекта или
	// клонирование). При успехе пересобираем главный экран вокруг нового
	// Repo, перенося уже полученную GitHub-авторизацию.
	case repoOpenedMsg:
		var ncmd tea.Cmd
		a.noRepo, ncmd = a.noRepo.Update(msg)
		if msg.err != nil {
			return a, ncmd
		}
		newMain := newMainModel(msg.repo)
		newMain.ghUser = a.main.ghUser
		newMain.ghToken = a.main.ghToken
		newMain.width, newMain.height = a.width, a.height
		a.main = newMain
		a.state = stateMain
		return a, tea.Batch(ncmd, a.main.Init())

	default:
		var lcmd, mcmd, ncmd tea.Cmd
		a.login, lcmd = a.login.Update(msg)
		a.main, mcmd = a.main.Update(msg)
		a.noRepo, ncmd = a.noRepo.Update(msg)
		return a, tea.Batch(lcmd, mcmd, ncmd)
	}
}

func (a App) View() string {
	switch a.state {
	case stateCheckingToken, stateLogin:
		return a.login.View()
	case stateNoRepo:
		return a.noRepo.View()
	default:
		return a.main.View()
	}
}
