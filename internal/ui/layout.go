package ui

// panelGeom — абсолютные экранные координаты одной панели главного экрана
// (0-индексация, как у tea.MouseMsg.X/Y), плюс диапазон строк со списком
// элементов. Используется и для рендера (contentW/contentH передаются в
// lipgloss .Width()/.Height()), и для попадания мыши — оба места опираются
// на один и тот же расчёт, чтобы клики не «съезжали» относительно рамок.
type panelGeom struct {
	colStart, colEnd int // [start, end), включая рамку
	rowStart, rowEnd int // [start, end), включая рамку

	contentW, contentH int // значения для .Width()/.Height() (внутри рамки, с учётом padding)

	itemsRowStart, itemsRowEnd int // [start, end) — строки со списком, без строки заголовка
}

func (g panelGeom) contains(x, y int) bool {
	return x >= g.colStart && x < g.colEnd && y >= g.rowStart && y < g.rowEnd
}

// itemIndexAt переводит абсолютную строку клика в индекс элемента списка,
// повторяя ту же логику скролл-окна (visibleWindow), что и рендер панели.
func (g panelGeom) itemIndexAt(y, cursor, total int) (int, bool) {
	if total == 0 || y < g.itemsRowStart || y >= g.itemsRowEnd {
		return 0, false
	}
	maxRows := g.itemsRowEnd - g.itemsRowStart
	start, _ := visibleWindow(cursor, total, maxRows)
	idx := start + (y - g.itemsRowStart)
	if idx < 0 || idx >= total {
		return 0, false
	}
	return idx, true
}

type normalLayout struct {
	branches, files, log, diff panelGeom
	toolbarRow, statusRow      int
}

// computeNormalLayout считает геометрию главного экрана (4 панели + тулбар +
// строка статуса) внутри внешнего Padding(1,2). Рамка каждой панели даёт +2
// к ширине/высоте сверх content-размера, переданного в .Width()/.Height() —
// см. эмпирическую проверку box-модели lipgloss (Border+Padding) в комментах
// ниже по коду рендера. Держим формулу в одном месте, чтобы рендер и
// hit-тестирование мыши никогда не разъезжались.
func computeNormalLayout(width, height int, hasErr bool) normalLayout {
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 32
	}

	topLines := 1
	if hasErr {
		topLines = 2
	}

	// hTotal — полная высота рамки (с бордером) каждой из 3 колонок тела.
	hTotal := height - 6 - topLines
	if hTotal < 10 {
		hTotal = 10
	}
	contentH := hTotal - 2
	if contentH < 3 {
		contentH = 3
	}
	// Колонка "Файлы"+"Лог" — те же hTotal снаружи, но два бордера вместо одного.
	stackContentH := contentH - 2
	if stackContentH < 2 {
		stackContentH = 2
	}
	topHalf := maxInt(stackContentH/2, 1)
	botHalf := maxInt(stackContentH-topHalf, 1)

	innerWidth := width - 4 // внешний Padding(1,2): 2 слева + 2 справа
	colContentSum := innerWidth - 6
	if colContentSum < 60 {
		colContentSum = 60
	}
	leftW := 26
	rightW := (colContentSum - leftW) * 4 / 10
	if rightW < 20 {
		rightW = 20
	}
	midW := colContentSum - leftW - rightW
	if midW < 20 {
		midW = 20
	}

	bodyTop := topLines + 2 // внешний pad(1) + строки шапки + пустая строка(1)
	bodyLeft := 2           // внешний pad слева

	branchesColStart := bodyLeft
	branchesColEnd := branchesColStart + leftW + 2
	midColStart := branchesColEnd
	midColEnd := midColStart + midW + 2
	diffColStart := midColEnd
	diffColEnd := diffColStart + rightW + 2

	branches := panelGeom{
		colStart: branchesColStart, colEnd: branchesColEnd,
		rowStart: bodyTop, rowEnd: bodyTop + hTotal,
		contentW: leftW, contentH: contentH,
		itemsRowStart: bodyTop + 2, itemsRowEnd: bodyTop + hTotal - 1,
	}
	diff := panelGeom{
		colStart: diffColStart, colEnd: diffColEnd,
		rowStart: bodyTop, rowEnd: bodyTop + hTotal,
		contentW: rightW, contentH: contentH,
		itemsRowStart: bodyTop + 2, itemsRowEnd: bodyTop + hTotal - 1,
	}

	filesRowStart := bodyTop
	filesRowEnd := filesRowStart + topHalf + 2
	files := panelGeom{
		colStart: midColStart, colEnd: midColEnd,
		rowStart: filesRowStart, rowEnd: filesRowEnd,
		contentW: midW, contentH: topHalf,
		itemsRowStart: filesRowStart + 2, itemsRowEnd: filesRowEnd - 1,
	}

	logRowStart := filesRowEnd
	logRowEnd := logRowStart + botHalf + 2
	log := panelGeom{
		colStart: midColStart, colEnd: midColEnd,
		rowStart: logRowStart, rowEnd: logRowEnd,
		contentW: midW, contentH: botHalf,
		itemsRowStart: logRowStart + 2, itemsRowEnd: logRowEnd - 1,
	}

	return normalLayout{
		branches:   branches,
		files:      files,
		log:        log,
		diff:       diff,
		toolbarRow: bodyTop + hTotal + 1,
		statusRow:  bodyTop + hTotal + 2,
	}
}
