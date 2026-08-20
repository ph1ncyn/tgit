package ui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tgit/internal/doctor"
	"tgit/internal/ghauth"
	"tgit/internal/gitrepo"
)

type mainMode int

const (
	modeNormal mainMode = iota
	modeCommitMsg
	modeBranchSwitch
	modeDoctorList
	modeDoctorConfirm
	modeStashList
	modeStashConfirmDrop
)

const (
	focusBranches = iota
	focusFiles
	focusLog
	focusDiff
)

type repoDataMsg struct {
	branch        string
	ahead, behind int
	branches      []string
	files         []gitrepo.FileStatus
	commits       []gitrepo.Commit
	err           error
}

type mainModel struct {
	repo    *gitrepo.Repo
	ghUser  *ghauth.User
	ghToken string

	branch        string
	ahead, behind int
	branches      []string
	branchCursor  int

	files      []gitrepo.FileStatus
	fileCursor int

	commits      []gitrepo.Commit
	commitCursor int

	diff       string
	diffScroll int

	mode  mainMode
	focus int

	commitInput       textinput.Model
	branchFilter      textinput.Model
	branchModalCursor int

	doctorIssues []doctor.Issue
	doctorCursor int

	stashes     []gitrepo.Stash
	stashCursor int
	stashDetail string

	busy      bool
	busyLabel string
	spinner   spinner.Model

	status    string
	statusErr bool
	err       string

	width, height int
}

func newMainModel(repo *gitrepo.Repo) mainModel {
	ci := textinput.New()
	ci.Placeholder = "сообщение коммита"
	ci.CharLimit = 500
	ci.Width = 60

	bf := textinput.New()
	bf.Placeholder = "фильтр веток / имя новой ветки"
	bf.Width = 40

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = titleStyle

	return mainModel{repo: repo, commitInput: ci, branchFilter: bf, spinner: sp, focus: focusFiles}
}

// IsModal сообщает, активен ли сейчас поверх основного экрана диалог с
// собственным вводом (коммит, переключатель веток, doctor) — в этом
// состоянии глобальные однобуквенные горячие клавиши (например 'g') не
// должны перехватываться и обязаны попадать в текстовое поле.
func (m mainModel) IsModal() bool {
	return m.mode != modeNormal
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
		ahead, behind, _ := repo.AheadBehind() // нет upstream — не ошибка, просто 0/0
		branches, err := repo.Branches()
		if err != nil {
			return repoDataMsg{err: err}
		}
		files, err := repo.Status()
		if err != nil {
			return repoDataMsg{err: err}
		}
		commits, err := repo.Log(100)
		if err != nil {
			return repoDataMsg{err: err}
		}
		return repoDataMsg{branch: branch, ahead: ahead, behind: behind, branches: branches, files: files, commits: commits}
	}
}

func (m mainModel) Init() tea.Cmd {
	return m.loadCmd()
}

func (m mainModel) Update(msg tea.Msg) (mainModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case spinner.TickMsg:
		if !m.busy {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case repoDataMsg:
		return m.applyRepoData(msg)

	case diffLoadedMsg:
		if msg.err != nil {
			m.diff = "не удалось получить diff: " + msg.err.Error()
		} else if msg.content == "" {
			m.diff = "(пусто)"
		} else {
			m.diff = msg.content
		}
		m.diffScroll = 0
		return m, nil

	case actionResultMsg:
		return m.applyActionResult(msg)

	case doctorScannedMsg:
		m.busy = false
		if msg.err != nil {
			m.status = "doctor: " + msg.err.Error()
			m.statusErr = true
			return m, nil
		}
		m.doctorIssues = msg.issues
		m.doctorCursor = 0
		m.mode = modeDoctorList
		return m, nil

	case doctorFixedMsg:
		m.busy = false
		if msg.err != nil {
			m.status = "не удалось исправить: " + msg.err.Error()
			m.statusErr = true
			m.mode = modeDoctorList
			return m, nil
		}
		m.status = "исправлено: " + msg.title
		m.statusErr = false
		m.mode = modeNormal
		return m, m.loadCmd()

	case stashListedMsg:
		m.busy = false
		if msg.err != nil {
			m.status = "стэш: " + msg.err.Error()
			m.statusErr = true
			return m, nil
		}
		m.stashes = msg.stashes
		m.stashCursor = clamp(m.stashCursor, 0, maxInt(len(m.stashes)-1, 0))
		m.stashDetail = ""
		m.mode = modeStashList
		return m, m.stashDetailCmd()

	case stashStatMsg:
		if msg.err != nil {
			m.stashDetail = "не удалось получить содержимое: " + msg.err.Error()
		} else {
			m.stashDetail = msg.content
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)
	}
	return m, nil
}

func (m mainModel) applyRepoData(msg repoDataMsg) (mainModel, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err.Error()
		return m, nil
	}
	m.err = ""
	m.branch = msg.branch
	m.ahead, m.behind = msg.ahead, msg.behind
	m.branches = msg.branches
	m.branchCursor = clamp(m.branchCursor, 0, maxInt(len(m.branches)-1, 0))
	m.files = msg.files
	m.fileCursor = clamp(m.fileCursor, 0, maxInt(len(m.files)-1, 0))
	m.commits = msg.commits
	m.commitCursor = clamp(m.commitCursor, 0, maxInt(len(m.commits)-1, 0))
	return m, m.selectionDiffCmd()
}

func (m mainModel) applyActionResult(msg actionResultMsg) (mainModel, tea.Cmd) {
	m.busy = false
	if msg.err != nil {
		m.status = actionLabel(msg.action) + ": " + msg.err.Error()
		m.statusErr = true
		return m, nil
	}
	m.status = actionSuccessText(msg.action, msg.arg, msg.output)
	m.statusErr = false
	// selectionDiffCmd не нужен отдельно: applyRepoData сам обновит diff,
	// когда придёт repoDataMsg от loadCmd.
	cmds := []tea.Cmd{m.loadCmd()}
	if strings.HasPrefix(msg.action, "stash-") && m.mode == modeStashList {
		// новый стэш создан прямо из модалки списка — обновляем и список тоже,
		// не закрывая её (в отличие от pop/apply/drop, которые сами уходят в modeNormal).
		cmds = append(cmds, stashListCmd(m.repo))
	}
	return m, tea.Batch(cmds...)
}

func actionLabel(action string) string {
	switch action {
	case "push":
		return "push"
	case "pull":
		return "pull"
	case "fetch":
		return "fetch"
	case "checkout":
		return "переключение ветки"
	case "create-branch":
		return "создание ветки"
	case "commit":
		return "коммит"
	case "stash-push":
		return "stash"
	case "stash-pop":
		return "stash pop"
	case "stash-apply":
		return "stash apply"
	case "stash-drop":
		return "stash drop"
	}
	return action
}

func actionSuccessText(action, arg, output string) string {
	switch action {
	case "push":
		if output == "" {
			return "push выполнен"
		}
		return "push: " + firstLine(output)
	case "pull":
		if output == "" {
			return "pull выполнен"
		}
		return "pull: " + firstLine(output)
	case "fetch":
		return "fetch выполнен"
	case "checkout":
		return "переключено на " + arg
	case "create-branch":
		return "создана и выбрана ветка " + arg
	case "commit":
		return "коммит создан"
	case "stash-push":
		return "изменения спрятаны в стэш"
	case "stash-pop":
		return "файлы возвращены из стэша"
	case "stash-apply":
		return "файлы применены из стэша (запись сохранена в стэше)"
	case "stash-drop":
		return "стэш удалён"
	}
	return "готово"
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// ---------- клавиши ----------

func (m mainModel) handleKey(msg tea.KeyMsg) (mainModel, tea.Cmd) {
	switch m.mode {
	case modeCommitMsg:
		return m.handleCommitKey(msg)
	case modeBranchSwitch:
		return m.handleBranchSwitchKey(msg)
	case modeDoctorList:
		return m.handleDoctorListKey(msg)
	case modeDoctorConfirm:
		return m.handleDoctorConfirmKey(msg)
	case modeStashList:
		return m.handleStashListKey(msg)
	case modeStashConfirmDrop:
		return m.handleStashConfirmDropKey(msg)
	default:
		return m.handleNormalKey(msg)
	}
}

func (m mainModel) handleNormalKey(msg tea.KeyMsg) (mainModel, tea.Cmd) {
	if m.busy {
		return m, nil
	}
	switch msg.String() {
	case "tab":
		m.focus = (m.focus + 1) % 4
		return m, m.selectionDiffCmd()
	case "shift+tab":
		m.focus = (m.focus + 3) % 4
		return m, m.selectionDiffCmd()
	case "up", "k":
		return m.moveCursor(-1)
	case "down", "j":
		return m.moveCursor(1)
	case "r":
		m.status = ""
		return m, m.loadCmd()
	case " ":
		return m.toggleStage()
	case "enter":
		return m.enterAction()
	case "y":
		if m.focus == focusLog && m.commitCursor < len(m.commits) {
			_ = clipboard.WriteAll(m.commits[m.commitCursor].Hash)
			m.status = "хеш скопирован в буфер обмена"
			m.statusErr = false
		}
		return m, nil
	case "c":
		return m.openCommitModal()
	case "b":
		if m.repo == nil {
			return m, nil
		}
		m.mode = modeBranchSwitch
		m.branchFilter.SetValue("")
		m.branchFilter.Focus()
		m.branchModalCursor = 0
		return m, textinput.Blink
	case "d":
		if m.repo == nil {
			m.status, m.statusErr = "нет репозитория", true
			return m, nil
		}
		m.busy, m.busyLabel = true, "проверяю репозиторий..."
		return m, tea.Batch(doctorScanCmd(m.repo), m.spinner.Tick)
	case "s":
		if m.repo == nil {
			return m, nil
		}
		m.busy, m.busyLabel = true, "загружаю стэш..."
		return m, tea.Batch(stashListCmd(m.repo), m.spinner.Tick)
	case "S":
		// быстрый pop последнего стэша прямо с главного экрана, без открытия
		// панели Stash — для выбора конкретной записи/apply/drop открой 's'.
		if m.repo == nil {
			return m, nil
		}
		m.busy, m.busyLabel = true, "достаю файлы из стэша..."
		return m, tea.Batch(stashPopCmd(m.repo, ""), m.spinner.Tick)
	case "p":
		if m.repo == nil {
			return m, nil
		}
		m.busy, m.busyLabel = true, "push..."
		return m, tea.Batch(pushCmd(m.repo, m.ghToken), m.spinner.Tick)
	case "P":
		if m.repo == nil {
			return m, nil
		}
		m.busy, m.busyLabel = true, "pull..."
		return m, tea.Batch(pullCmd(m.repo, m.ghToken), m.spinner.Tick)
	case "f":
		if m.repo == nil {
			return m, nil
		}
		m.busy, m.busyLabel = true, "fetch..."
		return m, tea.Batch(fetchCmd(m.repo, m.ghToken), m.spinner.Tick)
	}
	return m, nil
}

func (m mainModel) moveCursor(delta int) (mainModel, tea.Cmd) {
	switch m.focus {
	case focusBranches:
		if len(m.branches) == 0 {
			return m, nil
		}
		m.branchCursor = clamp(m.branchCursor+delta, 0, len(m.branches)-1)
	case focusFiles:
		if len(m.files) == 0 {
			return m, nil
		}
		m.fileCursor = clamp(m.fileCursor+delta, 0, len(m.files)-1)
		return m, m.selectionDiffCmd()
	case focusLog:
		if len(m.commits) == 0 {
			return m, nil
		}
		m.commitCursor = clamp(m.commitCursor+delta, 0, len(m.commits)-1)
		return m, m.selectionDiffCmd()
	case focusDiff:
		maxScroll := maxInt(len(strings.Split(m.diff, "\n"))-1, 0)
		m.diffScroll = clamp(m.diffScroll+delta, 0, maxScroll)
	}
	return m, nil
}

func (m mainModel) selectionDiffCmd() tea.Cmd {
	if m.repo == nil {
		return nil
	}
	switch m.focus {
	case focusFiles:
		if m.fileCursor >= len(m.files) {
			return nil
		}
		f := m.files[m.fileCursor]
		return diffFileCmd(m.repo, f.Path, f.Staged())
	case focusLog:
		if m.commitCursor >= len(m.commits) {
			return nil
		}
		return diffCommitCmd(m.repo, m.commits[m.commitCursor].Hash)
	}
	return nil
}

func (m mainModel) toggleStage() (mainModel, tea.Cmd) {
	if m.repo == nil || m.focus != focusFiles || m.fileCursor >= len(m.files) {
		return m, nil
	}
	f := m.files[m.fileCursor]
	var err error
	if f.Staged() {
		err = m.repo.UnstageFile(f.Path)
	} else {
		err = m.repo.StageFile(f.Path)
	}
	if err != nil {
		m.status, m.statusErr = "не удалось изменить индекс: "+err.Error(), true
		return m, nil
	}
	m.status = ""
	return m, m.loadCmd()
}

func (m mainModel) enterAction() (mainModel, tea.Cmd) {
	switch m.focus {
	case focusBranches:
		if m.repo == nil || m.branchCursor >= len(m.branches) {
			return m, nil
		}
		target := m.branches[m.branchCursor]
		if target == m.branch {
			return m, nil
		}
		m.busy, m.busyLabel = true, "переключаю ветку..."
		return m, tea.Batch(checkoutCmd(m.repo, target), m.spinner.Tick)
	case focusFiles:
		return m.toggleStage()
	}
	return m, nil
}

func (m mainModel) openCommitModal() (mainModel, tea.Cmd) {
	if m.repo == nil {
		return m, nil
	}
	hasStaged := false
	for _, f := range m.files {
		if f.Staged() {
			hasStaged = true
			break
		}
	}
	if !hasStaged {
		m.status, m.statusErr = "нечего коммитить — застейджите файлы (space)", true
		return m, nil
	}
	m.mode = modeCommitMsg
	m.commitInput.SetValue("")
	m.commitInput.Focus()
	return m, textinput.Blink
}

func (m mainModel) handleCommitKey(msg tea.KeyMsg) (mainModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = modeNormal
		m.commitInput.Blur()
		return m, nil
	case tea.KeyEnter:
		message := strings.TrimSpace(m.commitInput.Value())
		if message == "" {
			m.status, m.statusErr = "введите сообщение коммита", true
			return m, nil
		}
		m.mode = modeNormal
		m.commitInput.Blur()
		m.busy, m.busyLabel = true, "коммичу..."
		return m, tea.Batch(commitCmd(m.repo, message), m.spinner.Tick)
	}
	var cmd tea.Cmd
	m.commitInput, cmd = m.commitInput.Update(msg)
	return m, cmd
}

func (m mainModel) filteredBranches() []string {
	q := strings.ToLower(strings.TrimSpace(m.branchFilter.Value()))
	if q == "" {
		return m.branches
	}
	var out []string
	for _, b := range m.branches {
		if strings.Contains(strings.ToLower(b), q) {
			out = append(out, b)
		}
	}
	return out
}

func (m mainModel) handleBranchSwitchKey(msg tea.KeyMsg) (mainModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = modeNormal
		m.branchFilter.Blur()
		return m, nil
	case tea.KeyUp:
		m.branchModalCursor = clamp(m.branchModalCursor-1, 0, maxInt(len(m.filteredBranches())-1, 0))
		return m, nil
	case tea.KeyDown:
		m.branchModalCursor = clamp(m.branchModalCursor+1, 0, maxInt(len(m.filteredBranches())-1, 0))
		return m, nil
	case tea.KeyEnter:
		matches := m.filteredBranches()
		m.mode = modeNormal
		m.branchFilter.Blur()
		if len(matches) > 0 && m.branchModalCursor < len(matches) {
			target := matches[m.branchModalCursor]
			if target == m.branch {
				return m, nil
			}
			m.busy, m.busyLabel = true, "переключаю ветку..."
			return m, tea.Batch(checkoutCmd(m.repo, target), m.spinner.Tick)
		}
		name := strings.TrimSpace(m.branchFilter.Value())
		if name == "" {
			return m, nil
		}
		m.busy, m.busyLabel = true, "создаю ветку..."
		return m, tea.Batch(createBranchCmd(m.repo, name), m.spinner.Tick)
	}
	var cmd tea.Cmd
	m.branchFilter, cmd = m.branchFilter.Update(msg)
	m.branchModalCursor = 0
	return m, cmd
}

func (m mainModel) handleDoctorListKey(msg tea.KeyMsg) (mainModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = modeNormal
	case "up", "k":
		m.doctorCursor = clamp(m.doctorCursor-1, 0, maxInt(len(m.doctorIssues)-1, 0))
	case "down", "j":
		m.doctorCursor = clamp(m.doctorCursor+1, 0, maxInt(len(m.doctorIssues)-1, 0))
	case "enter", "f":
		if len(m.doctorIssues) > 0 {
			m.mode = modeDoctorConfirm
		}
	}
	return m, nil
}

func (m mainModel) handleDoctorConfirmKey(msg tea.KeyMsg) (mainModel, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if m.doctorCursor >= len(m.doctorIssues) {
			m.mode = modeDoctorList
			return m, nil
		}
		issue := m.doctorIssues[m.doctorCursor]
		m.busy, m.busyLabel = true, "исправляю..."
		return m, tea.Batch(doctorFixCmd(m.repo, issue), m.spinner.Tick)
	case "n", "esc":
		m.mode = modeDoctorList
	}
	return m, nil
}

func (m mainModel) stashDetailCmd() tea.Cmd {
	if m.repo == nil || m.stashCursor >= len(m.stashes) {
		return nil
	}
	return stashStatCmd(m.repo, m.stashes[m.stashCursor].Ref)
}

func (m mainModel) handleStashListKey(msg tea.KeyMsg) (mainModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = modeNormal
	case "up", "k":
		if len(m.stashes) == 0 {
			return m, nil
		}
		m.stashCursor = clamp(m.stashCursor-1, 0, len(m.stashes)-1)
		m.stashDetail = ""
		return m, m.stashDetailCmd()
	case "down", "j":
		if len(m.stashes) == 0 {
			return m, nil
		}
		m.stashCursor = clamp(m.stashCursor+1, 0, len(m.stashes)-1)
		m.stashDetail = ""
		return m, m.stashDetailCmd()
	case "n":
		if m.repo == nil {
			return m, nil
		}
		m.busy, m.busyLabel = true, "прячу изменения в стэш..."
		return m, tea.Batch(stashPushCmd(m.repo), m.spinner.Tick)
	case "enter", "p":
		if m.stashCursor >= len(m.stashes) {
			return m, nil
		}
		ref := m.stashes[m.stashCursor].Ref
		m.mode = modeNormal
		m.busy, m.busyLabel = true, "достаю файлы из стэша..."
		return m, tea.Batch(stashPopCmd(m.repo, ref), m.spinner.Tick)
	case "a":
		if m.stashCursor >= len(m.stashes) {
			return m, nil
		}
		ref := m.stashes[m.stashCursor].Ref
		m.mode = modeNormal
		m.busy, m.busyLabel = true, "применяю стэш..."
		return m, tea.Batch(stashApplyCmd(m.repo, ref), m.spinner.Tick)
	case "x":
		if m.stashCursor < len(m.stashes) {
			m.mode = modeStashConfirmDrop
		}
	}
	return m, nil
}

func (m mainModel) handleStashConfirmDropKey(msg tea.KeyMsg) (mainModel, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if m.stashCursor >= len(m.stashes) {
			m.mode = modeStashList
			return m, nil
		}
		ref := m.stashes[m.stashCursor].Ref
		m.mode = modeNormal
		m.busy, m.busyLabel = true, "удаляю стэш..."
		return m, tea.Batch(stashDropCmd(m.repo, ref), m.spinner.Tick)
	case "n", "esc":
		m.mode = modeStashList
	}
	return m, nil
}

// ---------- рендер ----------

func (m mainModel) View() string {
	switch m.mode {
	case modeCommitMsg:
		return m.viewCommitModal()
	case modeBranchSwitch:
		return m.viewBranchSwitchModal()
	case modeDoctorList, modeDoctorConfirm:
		return m.viewDoctorModal()
	case modeStashList, modeStashConfirmDrop:
		return m.viewStashModal()
	default:
		return m.viewNormal()
	}
}

func (m mainModel) viewNormal() string {
	layout := computeNormalLayout(m.width, m.height, m.err != "")

	top := m.renderTopBar()

	branchesPanel := m.renderBranchesPanel(layout.branches.contentW, layout.branches.contentH)
	filesPanel := m.renderFilesPanel(layout.files.contentW, layout.files.contentH)
	logPanel := m.renderLogPanel(layout.log.contentW, layout.log.contentH)
	diffPanel := m.renderDiffPanel(layout.diff.contentW, layout.diff.contentH)

	middle := lipgloss.JoinVertical(lipgloss.Left, filesPanel, logPanel)
	body := lipgloss.JoinHorizontal(lipgloss.Top, branchesPanel, middle, diffPanel)

	return lipgloss.NewStyle().Padding(1, 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, top, "", body, "", m.renderToolbar(), m.renderStatusLine()),
	)
}

func (m mainModel) renderTopBar() string {
	branchLabel := m.branch
	if branchLabel == "" {
		branchLabel = "—"
	}
	aheadBehind := ""
	if m.ahead > 0 || m.behind > 0 {
		aheadBehind = fmt.Sprintf(" ↑%d ↓%d", m.ahead, m.behind)
	}
	ghStatus := helpStyle.Render("GitHub: не подключён (g — войти)")
	if m.ghUser != nil {
		label := m.ghUser.Login
		if m.ghUser.Name != "" {
			label = fmt.Sprintf("%s (@%s)", m.ghUser.Name, m.ghUser.Login)
		}
		ghStatus = okStyle.Render("GitHub: " + label)
	}
	line := titleStyle.Render("tgit") + "   " + helpStyle.Render("⎇ "+branchLabel+aheadBehind) + "   " + ghStatus
	if m.err != "" {
		line += "\n" + errorStyle.Render(m.err)
	}
	return line
}

func (m mainModel) renderBranchesPanel(w, h int) string {
	maxRows := maxInt(h-1, 1) // h — content-высота (без рамки); строка 0 — заголовок, остальные h-1 — элементы
	innerW := maxInt(w-2, 1)  // w — content-ширина; 1 колонка padding с каждой стороны
	var lines []string
	if len(m.branches) == 0 {
		lines = append(lines, helpStyle.Render("нет данных"))
	} else {
		start, end := visibleWindow(m.branchCursor, len(m.branches), maxRows)
		for i := start; i < end; i++ {
			b := m.branches[i]
			prefix := "  "
			style := lipgloss.NewStyle()
			if b == m.branch {
				prefix, style = "⎇ ", okStyle
			}
			text := truncateLine(prefix+b, innerW)
			if i == m.branchCursor && m.focus == focusBranches {
				style = style.Reverse(true)
			}
			lines = append(lines, style.Render(text))
		}
	}
	title := titleStyle.Render("Ветки")
	return panelStyleFor(m.focus == focusBranches).Width(w).Height(h).Render(title + "\n" + strings.Join(lines, "\n"))
}

func (m mainModel) renderFilesPanel(w, h int) string {
	maxRows := maxInt(h-1, 1) // h — content-высота (без рамки); строка 0 — заголовок, остальные h-1 — элементы
	innerW := maxInt(w-2, 1)  // w — content-ширина; 1 колонка padding с каждой стороны
	var lines []string
	if len(m.files) == 0 {
		lines = append(lines, okStyle.Render("чисто"))
	} else {
		start, end := visibleWindow(m.fileCursor, len(m.files), maxRows)
		for i := start; i < end; i++ {
			f := m.files[i]
			mark := "[ ]"
			if f.Staged() {
				mark = "[x]"
			}
			text := truncateLine(fmt.Sprintf("%s %c%c %s", mark, f.X, f.Y, f.Path), innerW)
			style := fileLineStyle(f)
			if i == m.fileCursor && m.focus == focusFiles {
				style = style.Reverse(true)
			}
			lines = append(lines, style.Render(text))
		}
	}
	title := titleStyle.Render(fmt.Sprintf("Файлы (%d)", len(m.files)))
	return panelStyleFor(m.focus == focusFiles).Width(w).Height(h).Render(title + "\n" + strings.Join(lines, "\n"))
}

func fileLineStyle(f gitrepo.FileStatus) lipgloss.Style {
	switch {
	case f.Staged():
		return okStyle
	case f.Untracked():
		return helpStyle
	default:
		return errorStyle
	}
}

func (m mainModel) renderLogPanel(w, h int) string {
	maxRows := maxInt(h-1, 1) // h — content-высота (без рамки); строка 0 — заголовок, остальные h-1 — элементы
	innerW := maxInt(w-2, 1)  // w — content-ширина; 1 колонка padding с каждой стороны
	var lines []string
	if len(m.commits) == 0 {
		lines = append(lines, helpStyle.Render("нет коммитов"))
	} else {
		start, end := visibleWindow(m.commitCursor, len(m.commits), maxRows)
		for i := start; i < end; i++ {
			c := m.commits[i]
			text := truncateLine(c.Short+"  "+c.Subject, innerW)
			style := lipgloss.NewStyle()
			if i == m.commitCursor && m.focus == focusLog {
				style = style.Reverse(true)
			}
			lines = append(lines, style.Render(text))
		}
	}
	title := titleStyle.Render("Лог")
	return panelStyleFor(m.focus == focusLog).Width(w).Height(h).Render(title + "\n" + strings.Join(lines, "\n"))
}

func (m mainModel) renderDiffPanel(w, h int) string {
	maxRows := maxInt(h-1, 1) // h — content-высота (без рамки); строка 0 — заголовок, остальные h-1 — элементы
	innerW := maxInt(w-2, 1)  // w — content-ширина; 1 колонка padding с каждой стороны
	content := m.diff
	if content == "" {
		content = helpStyle.Render("выберите файл или коммит")
	}
	lines := strings.Split(content, "\n")
	scroll := clamp(m.diffScroll, 0, maxInt(len(lines)-maxRows, 0))
	end := scroll + maxRows
	if end > len(lines) {
		end = len(lines)
	}
	var out []string
	for _, l := range lines[scroll:end] {
		out = append(out, styleDiffLine(truncateLine(l, innerW)))
	}
	title := titleStyle.Render("Diff")
	return panelStyleFor(m.focus == focusDiff).Width(w).Height(h).Render(title + "\n" + strings.Join(out, "\n"))
}

func styleDiffLine(l string) string {
	switch {
	case strings.HasPrefix(l, "+++") || strings.HasPrefix(l, "---"):
		return helpStyle.Render(l)
	case strings.HasPrefix(l, "+"):
		return okStyle.Render(l)
	case strings.HasPrefix(l, "-"):
		return errorStyle.Render(l)
	case strings.HasPrefix(l, "@@"):
		return linkStyle.Render(l)
	default:
		return l
	}
}

func (m mainModel) renderToolbar() string {
	buttons := toolbarButtons()
	labels := make([]string, len(buttons))
	for i, b := range buttons {
		labels[i] = b.label
	}
	return helpStyle.Render(strings.Join(labels, "   "))
}

func (m mainModel) renderStatusLine() string {
	if m.busy {
		return titleStyle.Render(m.spinner.View()) + " " + helpStyle.Render(m.busyLabel)
	}
	if m.status == "" {
		return helpStyle.Render("tab — след. панель  •  ↑/↓ — навигация  •  space/enter — стейдж/чекаут  •  y — копировать хеш  •  ctrl+c — выйти")
	}
	style := okStyle
	if m.statusErr {
		style = errorStyle
	}
	return style.Render(m.status)
}

func (m mainModel) viewCommitModal() string {
	staged := 0
	for _, f := range m.files {
		if f.Staged() {
			staged++
		}
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("Новый коммит") + "\n\n")
	b.WriteString(helpStyle.Render(fmt.Sprintf("Застейджено файлов: %d", staged)) + "\n\n")
	b.WriteString("Сообщение: " + m.commitInput.View() + "\n\n")
	b.WriteString(helpStyle.Render("enter — закоммитить  •  esc — отмена"))
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (m mainModel) viewBranchSwitchModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Переключить / создать ветку") + "\n\n")
	b.WriteString("Фильтр: " + m.branchFilter.View() + "\n\n")

	matches := m.filteredBranches()
	if len(matches) == 0 {
		q := strings.TrimSpace(m.branchFilter.Value())
		if q == "" {
			b.WriteString(helpStyle.Render("нет веток"))
		} else {
			b.WriteString(helpStyle.Render("совпадений нет — enter создаст новую ветку «" + q + "»"))
		}
	} else {
		for i, br := range matches {
			prefix := "  "
			if br == m.branch {
				prefix = "⎇ "
			}
			style := lipgloss.NewStyle()
			if i == m.branchModalCursor {
				style = style.Reverse(true)
			}
			b.WriteString(style.Render(prefix+br) + "\n")
		}
	}
	b.WriteString("\n" + helpStyle.Render("enter — выбрать/создать  •  esc — отмена"))
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (m mainModel) viewDoctorModal() string {
	var b strings.Builder
	if len(m.doctorIssues) == 0 {
		b.WriteString(okStyle.Render("✓ Проблем не найдено") + "\n\n")
		b.WriteString(helpStyle.Render("esc — закрыть"))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	b.WriteString(errorStyle.Render(fmt.Sprintf("⚠ Найдено проблем: %d", len(m.doctorIssues))) + "\n\n")
	// Каждая проблема — ровно одна строка (без вставок под курсором): так
	// строки кликабельны мышью по фиксированному номеру, деталь курсора — ниже.
	for i, issue := range m.doctorIssues {
		marker := "  "
		style := lipgloss.NewStyle()
		if i == m.doctorCursor {
			marker, style = "> ", style.Bold(true)
		}
		b.WriteString(style.Render(marker+issue.Title) + "\n")
	}

	if m.doctorCursor < len(m.doctorIssues) {
		b.WriteString("\n" + helpStyle.Render(truncateLine(m.doctorIssues[m.doctorCursor].Detail, 90)) + "\n")
	}

	b.WriteString("\n")
	if m.mode == modeDoctorConfirm {
		b.WriteString(errorStyle.Render("Исправить эту проблему? (может удалить файлы / изменить .gitignore)") + "\n")
		b.WriteString(helpStyle.Render("y — да  •  n/esc — отмена"))
	} else {
		b.WriteString(helpStyle.Render("↑/↓ — выбрать  •  enter — исправить  •  esc — закрыть"))
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (m mainModel) viewStashModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Стэш") + "\n\n")

	if len(m.stashes) == 0 {
		b.WriteString(helpStyle.Render("стэш пуст") + "\n\n")
		b.WriteString(helpStyle.Render("n — спрятать текущие изменения в стэш  •  esc — закрыть"))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	for i, s := range m.stashes {
		marker := "  "
		style := lipgloss.NewStyle()
		if i == m.stashCursor {
			marker, style = "> ", style.Bold(true)
		}
		b.WriteString(style.Render(marker+s.Ref+"  "+s.Subject) + "\n")
	}

	if m.stashDetail != "" {
		b.WriteString("\n" + helpStyle.Render("Изменённые файлы:") + "\n")
		for _, l := range strings.Split(strings.TrimRight(m.stashDetail, "\n"), "\n") {
			b.WriteString(helpStyle.Render("  "+truncateLine(l, 100)) + "\n")
		}
	}

	b.WriteString("\n")
	if m.mode == modeStashConfirmDrop {
		b.WriteString(errorStyle.Render("Удалить этот стэш без применения? Изменения будут потеряны безвозвратно.") + "\n")
		b.WriteString(helpStyle.Render("y — да  •  n/esc — отмена"))
	} else {
		b.WriteString(helpStyle.Render("↑/↓ — выбрать  •  enter/p — pop (достать и удалить)  •  a — apply (достать, оставить в стэше)  •  x — drop (удалить)  •  n — новый стэш  •  esc — закрыть"))
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

// ---------- утилиты ----------

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func truncateLine(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

func visibleWindow(cursor, total, maxRows int) (start, end int) {
	if total <= maxRows {
		return 0, total
	}
	start = cursor - maxRows/2
	if start < 0 {
		start = 0
	}
	end = start + maxRows
	if end > total {
		end = total
		start = end - maxRows
	}
	if start < 0 {
		start = 0
	}
	return start, end
}
