package ui

import tea "github.com/charmbracelet/bubbletea"

// toolbarButton связывает подпись кнопки с клавишей, которую она эмулирует —
// клик по кнопке проигрывает то же самое событие клавиатуры, так что вся
// логика действия (busy/spinner/проверки) остаётся в одном месте.
type toolbarButton struct {
	label string
	key   string
}

func toolbarButtons() []toolbarButton {
	return []toolbarButton{
		{"[c] Commit", "c"},
		{"[p] Push", "p"},
		{"[P] Pull", "P"},
		{"[f] Fetch", "f"},
		{"[b] Branch", "b"},
		{"[s] Stash", "s"},
		{"[S] Pop", "S"},
		{"[d] Doctor", "d"},
	}
}

// toolbarHit определяет, на какую кнопку тулбара пришёлся клик по колонке x
// в строке toolbarRow.
func toolbarHit(x, toolbarRow, clickY int) (string, bool) {
	if clickY != toolbarRow {
		return "", false
	}
	const separatorWidth = 3 // "   " между кнопками, см. renderToolbar
	col := x - 2             // внешний Padding(1,2) слева
	if col < 0 {
		return "", false
	}
	for _, b := range toolbarButtons() {
		w := len([]rune(b.label))
		if col < w {
			return b.key, true
		}
		col -= w + separatorWidth
		if col < 0 {
			return "", false // клик пришёлся на промежуток между кнопками
		}
	}
	return "", false
}

func (m mainModel) handleMouse(msg tea.MouseMsg) (mainModel, tea.Cmd) {
	switch m.mode {
	case modeNormal:
		return m.handleNormalMouse(msg)
	case modeBranchSwitch:
		return m.handleBranchSwitchMouse(msg)
	case modeDoctorList:
		return m.handleDoctorMouse(msg)
	case modeStashList:
		return m.handleStashMouse(msg)
	}
	return m, nil
}

func (m mainModel) handleNormalMouse(msg tea.MouseMsg) (mainModel, tea.Cmd) {
	if m.busy {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.mouseScroll(msg.X, msg.Y, -1)
	case tea.MouseButtonWheelDown:
		return m.mouseScroll(msg.X, msg.Y, 1)
	}

	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	layout := computeNormalLayout(m.width, m.height, m.err != "")

	if key, ok := toolbarHit(msg.X, layout.toolbarRow, msg.Y); ok {
		return m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	}

	switch {
	case layout.branches.contains(msg.X, msg.Y):
		m.focus = focusBranches
		if idx, ok := layout.branches.itemIndexAt(msg.Y, m.branchCursor, len(m.branches)); ok {
			m.branchCursor = idx
		}
		return m, nil

	case layout.files.contains(msg.X, msg.Y):
		m.focus = focusFiles
		if idx, ok := layout.files.itemIndexAt(msg.Y, m.fileCursor, len(m.files)); ok {
			m.fileCursor = idx
			return m, m.selectionDiffCmd()
		}
		return m, nil

	case layout.log.contains(msg.X, msg.Y):
		m.focus = focusLog
		if idx, ok := layout.log.itemIndexAt(msg.Y, m.commitCursor, len(m.commits)); ok {
			m.commitCursor = idx
			return m, m.selectionDiffCmd()
		}
		return m, nil

	case layout.diff.contains(msg.X, msg.Y):
		m.focus = focusDiff
		return m, nil
	}

	return m, nil
}

// mouseScroll наводит фокус на панель под курсором и крутит её тем же
// moveCursor, что и клавиши ↑/↓ — колесо мыши работает независимо от того,
// какая панель была активна с клавиатуры.
func (m mainModel) mouseScroll(x, y, delta int) (mainModel, tea.Cmd) {
	layout := computeNormalLayout(m.width, m.height, m.err != "")
	switch {
	case layout.branches.contains(x, y):
		m.focus = focusBranches
	case layout.files.contains(x, y):
		m.focus = focusFiles
	case layout.log.contains(x, y):
		m.focus = focusLog
	case layout.diff.contains(x, y):
		m.focus = focusDiff
	default:
		return m, nil
	}
	return m.moveCursor(delta)
}

// ---------- модалки: клик/колесо по списку без рамок lipgloss, координаты
// считаются напрямую от порядка строк в тексте (см. view* соответствующей модалки).

func (m mainModel) handleBranchSwitchMouse(msg tea.MouseMsg) (mainModel, tea.Cmd) {
	matches := m.filteredBranches()
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.branchModalCursor = clamp(m.branchModalCursor-1, 0, maxInt(len(matches)-1, 0))
		return m, nil
	case tea.MouseButtonWheelDown:
		m.branchModalCursor = clamp(m.branchModalCursor+1, 0, maxInt(len(matches)-1, 0))
		return m, nil
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	const headerLines = 4 // заголовок(1)+пусто(1)+"Фильтр: ..."(1)+пусто(1), см. viewBranchSwitchModal
	idx := msg.Y - 1 - headerLines
	if idx >= 0 && idx < len(matches) {
		m.branchModalCursor = idx
	}
	return m, nil
}

func (m mainModel) handleDoctorMouse(msg tea.MouseMsg) (mainModel, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.doctorCursor = clamp(m.doctorCursor-1, 0, maxInt(len(m.doctorIssues)-1, 0))
		return m, nil
	case tea.MouseButtonWheelDown:
		m.doctorCursor = clamp(m.doctorCursor+1, 0, maxInt(len(m.doctorIssues)-1, 0))
		return m, nil
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	const headerLines = 2 // заголовок(1)+пусто(1), см. viewDoctorModal
	idx := msg.Y - 1 - headerLines
	if idx >= 0 && idx < len(m.doctorIssues) {
		m.doctorCursor = idx
	}
	return m, nil
}

func (m mainModel) handleStashMouse(msg tea.MouseMsg) (mainModel, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if len(m.stashes) == 0 {
			return m, nil
		}
		m.stashCursor = clamp(m.stashCursor-1, 0, len(m.stashes)-1)
		m.stashDetail = ""
		return m, m.stashDetailCmd()
	case tea.MouseButtonWheelDown:
		if len(m.stashes) == 0 {
			return m, nil
		}
		m.stashCursor = clamp(m.stashCursor+1, 0, len(m.stashes)-1)
		m.stashDetail = ""
		return m, m.stashDetailCmd()
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	const headerLines = 2 // заголовок(1)+пусто(1), см. viewStashModal
	idx := msg.Y - 1 - headerLines
	if idx >= 0 && idx < len(m.stashes) {
		m.stashCursor = idx
		m.stashDetail = ""
		return m, m.stashDetailCmd()
	}
	return m, nil
}
