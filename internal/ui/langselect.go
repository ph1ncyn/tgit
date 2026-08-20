package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tgit/internal/i18n"
)

// langOption pairs a selectable menu row with the language it activates.
type langOption struct {
	lang  i18n.Lang
	label string
}

var langOptions = []langOption{
	{i18n.English, "English"},
	{i18n.Russian, "Русский"},
}

// langModel is the very first screen tgit shows: pick an interface language
// before anything else (including the token screen) renders. It always
// displays both language names regardless of the language chosen so far,
// since that choice hasn't been made yet.
type langModel struct {
	cursor int
	width  int
}

func newLangModel() langModel {
	return langModel{}
}

func (m langModel) Init() tea.Cmd { return nil }

func (m langModel) Update(msg tea.Msg) (langModel, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsz.Width
	}
	return m, nil
}

// handleKey moves the cursor or, on enter, reports the chosen language via
// the boolean return so the caller (App.Update) can switch screens.
func (m langModel) handleKey(msg tea.KeyMsg) (langModel, i18n.Lang, bool) {
	switch msg.String() {
	case "up", "k":
		m.cursor = clamp(m.cursor-1, 0, len(langOptions)-1)
	case "down", "j":
		m.cursor = clamp(m.cursor+1, 0, len(langOptions)-1)
	case "enter":
		return m, langOptions[m.cursor].lang, true
	}
	return m, i18n.English, false
}

func (m langModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("tgit") + "\n\n")
	b.WriteString("Select interface language / Выберите язык интерфейса:\n\n")

	for i, opt := range langOptions {
		prefix := "  "
		style := lipgloss.NewStyle()
		if i == m.cursor {
			prefix, style = "> ", style.Reverse(true)
		}
		b.WriteString(style.Render(prefix+opt.label) + "\n")
	}

	b.WriteString("\n" + helpStyle.Render("↑/↓ — select / выбрать  •  enter — confirm / подтвердить  •  ctrl+c — quit"))
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
