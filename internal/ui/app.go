package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"tgit/internal/config"
	"tgit/internal/gitrepo"
	"tgit/internal/i18n"
	"tgit/internal/secret"
)

type screenState int

const (
	stateLang screenState = iota
	stateCheckingToken
	stateLogin
	stateMain
	stateNoRepo
)

// App — корневая модель Bubble Tea, переключающая экран выбора языка, экран
// входа, главный экран и экран "репозиторий не найден".
type App struct {
	state  screenState
	lang   langModel
	login  loginModel
	main   mainModel
	noRepo noRepoModel

	pendingToken   string
	afterLangState screenState
	width, height  int
}

// NewApp собирает приложение. Первый экран — выбор языка интерфейса, но
// только пока язык ни разу не был выбран: как только пользователь один раз
// подтверждает его на этом экране, выбор сохраняется на диск (см.
// applyLanguage/config.SaveLanguage) и при следующих запусках этот экран
// пропускается — язык применяется сразу, до отрисовки первого кадра.
// После выбора языка: если в системном хранилище уже есть токен, приложение
// тихо проверит его перед показом главного экрана; если токена нет — сразу
// показывает экран входа со ссылкой на создание токена. Если repo не
// открылся (текущая папка — не git-репозиторий), после экрана входа
// показывается не главный экран, а stateNoRepo.
func NewApp(repo *gitrepo.Repo, cwd string) App {
	a := App{
		lang:   newLangModel(),
		login:  newLoginModel(),
		main:   newMainModel(repo),
		noRepo: newNoRepoModel(cwd),
	}
	if tok, err := secret.Load(); err == nil && tok != "" {
		a.afterLangState = stateCheckingToken
		a.pendingToken = tok
		a.login.checking = true
	} else {
		a.afterLangState = stateLogin
	}

	if code, ok := config.LoadLanguage(); ok {
		if lang, ok := i18n.ParseCode(code); ok {
			a.applyLanguage(lang)
			a.state = a.afterLangState
			return a
		}
	}
	a.state = stateLang
	return a
}

// applyLanguage делает lang активным языком интерфейса. commitInput и
// branchFilter получили плейсхолдеры от newMainModel ещё при сборке main
// (до того, как язык мог быть известен), поэтому обновляем их текст здесь
// же, а не полагаемся на повторное чтение i18n.T при следующем рендере.
func (a *App) applyLanguage(lang i18n.Lang) {
	i18n.Set(lang)
	a.main.commitInput.Placeholder = i18n.T.CommitInputPlaceholder
	a.main.branchFilter.Placeholder = i18n.T.BranchFilterPlaceholder
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
	cmds := []tea.Cmd{a.main.Init(), a.noRepo.Init(), a.lang.Init()}
	switch a.state {
	// Если язык ещё предстоит выбрать (stateLang), validateCmd/login.Init() не
	// запускаем здесь — эти команды откладываются до подтверждения языка (см.
	// stateLang в Update), иначе тихая проверка токена могла бы отрендериться
	// раньше, чем пользователь успеет выбрать язык. Если же язык уже был
	// сохранён с прошлого запуска (см. NewApp), состояние стартует сразу с
	// afterLangState, и эти команды нужно запустить прямо тут — иначе главный
	// экран/логин повиснут без данных.
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
		var lgcmd, lcmd, mcmd, ncmd tea.Cmd
		a.lang, lgcmd = a.lang.Update(msg)
		a.login, lcmd = a.login.Update(msg)
		a.main, mcmd = a.main.Update(msg)
		a.noRepo, ncmd = a.noRepo.Update(msg)
		return a, tea.Batch(lgcmd, lcmd, mcmd, ncmd)

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return a, tea.Quit
		}
		switch a.state {
		case stateLang:
			var chosen i18n.Lang
			var confirmed bool
			a.lang, chosen, confirmed = a.lang.handleKey(msg)
			if !confirmed {
				return a, nil
			}
			a.applyLanguage(chosen)
			_ = config.SaveLanguage(chosen.Code())
			a.state = a.afterLangState
			switch a.state {
			case stateCheckingToken:
				return a, tea.Batch(validateCmd(a.pendingToken), a.login.spinner.Tick)
			case stateLogin:
				return a, a.login.Init()
			}
			return a, nil

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
				a.login.errMsg = i18n.T.SavedTokenInvalid
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
		var lgcmd, lcmd, mcmd, ncmd tea.Cmd
		a.lang, lgcmd = a.lang.Update(msg)
		a.login, lcmd = a.login.Update(msg)
		a.main, mcmd = a.main.Update(msg)
		a.noRepo, ncmd = a.noRepo.Update(msg)
		return a, tea.Batch(lgcmd, lcmd, mcmd, ncmd)
	}
}

func (a App) View() string {
	switch a.state {
	case stateLang:
		return a.lang.View()
	case stateCheckingToken, stateLogin:
		return a.login.View()
	case stateNoRepo:
		return a.noRepo.View()
	default:
		return a.main.View()
	}
}
