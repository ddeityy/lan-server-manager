package ui

import (
	"cmp"
	"fmt"
	"image/color"
	"slices"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/assets"
	"lan-server-manager/game/logparse"
	"lan-server-manager/game/scoreboard"
)

const scoreboardCellGap = float32(4)

// Column titles used as identifiers for special rendering/sorting.
const (
	colClass = "Class"
	colKick  = "Kick"
	colKills = "K"
	colName  = "Name"
)

type scoreboardColumn struct {
	Title string
	Width float32
	Fmt   func(p scoreboard.PlayerStats, elapsed time.Duration) string
	Less  func(a, b scoreboard.PlayerStats, elapsed time.Duration) bool
}

// lessOrdered returns a column Less function that orders rows by the value
// extracted with get. It works for any [cmp.Ordered] type (int, float64, string, etc.).
func lessOrdered[T cmp.Ordered](get func(p scoreboard.PlayerStats, elapsed time.Duration) T) func(a, b scoreboard.PlayerStats, elapsed time.Duration) bool {
	return func(a, b scoreboard.PlayerStats, elapsed time.Duration) bool {
		return get(a, elapsed) < get(b, elapsed)
	}
}

const (
	nameColumnWidth   = float32(170)
	narrowColumnWidth = float32(40)
	mediumColumnWidth = float32(55)
	dmgColumnWidth    = float32(65)
	dtColumnWidth     = float32(60)
	kickColumnWidth   = float32(55)
	scoreboardRowGap  = float32(2)
	titleBarHeight    = float32(28)
	titleBarMinWidth  = float32(200)
	classOrderUnknown = 99
)

var scoreboardColumns = []scoreboardColumn{
	{
		Title: colName,
		Width: nameColumnWidth,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return p.Name },
		Less:  lessOrdered(func(p scoreboard.PlayerStats, _ time.Duration) string { return p.Name }),
	},
	{
		Title: colClass,
		Width: narrowColumnWidth,
		Fmt:   func(_ scoreboard.PlayerStats, _ time.Duration) string { return "" },
		Less:  lessOrdered(func(p scoreboard.PlayerStats, _ time.Duration) int { return classOrder(p.Class) }),
	},
	{
		Title: colKills,
		Width: narrowColumnWidth,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return strconv.Itoa(p.Kills) },
		Less:  lessOrdered(func(p scoreboard.PlayerStats, _ time.Duration) int { return p.Kills }),
	},
	{
		Title: "A",
		Width: narrowColumnWidth,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return strconv.Itoa(p.Assists) },
		Less:  lessOrdered(func(p scoreboard.PlayerStats, _ time.Duration) int { return p.Assists }),
	},
	{
		Title: "D",
		Width: narrowColumnWidth,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return strconv.Itoa(p.Deaths) },
		Less:  lessOrdered(func(p scoreboard.PlayerStats, _ time.Duration) int { return p.Deaths }),
	},
	{
		Title: "DMG",
		Width: dmgColumnWidth,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return strconv.Itoa(p.Damage) },
		Less:  lessOrdered(func(p scoreboard.PlayerStats, _ time.Duration) int { return p.Damage }),
	},
	{
		Title: "DPM",
		Width: mediumColumnWidth,
		Fmt: func(p scoreboard.PlayerStats, elapsed time.Duration) string {
			return fmt.Sprintf("%.0f", p.DPM(elapsed))
		},
		Less: lessOrdered(func(p scoreboard.PlayerStats, elapsed time.Duration) float64 { return p.DPM(elapsed) }),
	},
	{
		Title: "DT",
		Width: dtColumnWidth,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return strconv.Itoa(p.DamageTaken) },
		Less:  lessOrdered(func(p scoreboard.PlayerStats, _ time.Duration) int { return p.DamageTaken }),
	},
	{
		Title: "DTM",
		Width: mediumColumnWidth,
		Fmt: func(p scoreboard.PlayerStats, elapsed time.Duration) string {
			return fmt.Sprintf("%.0f", p.DTM(elapsed))
		},
		Less: lessOrdered(func(p scoreboard.PlayerStats, elapsed time.Duration) float64 { return p.DTM(elapsed) }),
	},
	{
		Title: "Heals",
		Width: mediumColumnWidth,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return strconv.Itoa(p.Heals) },
		Less:  lessOrdered(func(p scoreboard.PlayerStats, _ time.Duration) int { return p.Heals }),
	},
	{
		Title: "Cap",
		Width: float32(35),
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return strconv.Itoa(p.Caps) },
		Less:  lessOrdered(func(p scoreboard.PlayerStats, _ time.Duration) int { return p.Caps }),
	},
	{
		Title: "KD",
		Width: float32(45),
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return fmt.Sprintf("%.2f", p.KD()) },
		Less:  lessOrdered(func(p scoreboard.PlayerStats, _ time.Duration) float64 { return p.KD() }),
	},
	{
		Title: "KAD",
		Width: mediumColumnWidth,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return fmt.Sprintf("%.2f", p.KAD()) },
		Less:  lessOrdered(func(p scoreboard.PlayerStats, _ time.Duration) float64 { return p.KAD() }),
	},
	{
		Title: "Ping",
		Width: float32(45),
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return strconv.Itoa(p.Ping) },
		Less:  lessOrdered(func(p scoreboard.PlayerStats, _ time.Duration) int { return p.Ping }),
	},
	{
		Title: colKick,
		Width: kickColumnWidth,
	},
}

var (
	redTeamColor = color.NRGBA{R: 180, G: 60, B: 60, A: 90}
	bluTeamColor = color.NRGBA{R: 60, G: 100, B: 180, A: 90}
)

// ScoreboardView renders two sortable team tables side by side, plus a small
// unassigned player area fed by the initial RCON status response.
type ScoreboardView struct {
	redTable   *teamTable
	bluTable   *teamTable
	unassigned *fyne.Container
	onKick     func(scoreboard.PlayerStats)
}

func newScoreboardView() *ScoreboardView {
	return &ScoreboardView{
		redTable:   newTeamTable("RED", redTeamColor, fyne.TextAlignLeading, false),
		bluTable:   newTeamTable("BLU", bluTeamColor, fyne.TextAlignTrailing, true),
		unassigned: container.NewVBox(),
	}
}

// SetOnKick configures the callback invoked when a row's kick button is tapped.
func (sv *ScoreboardView) SetOnKick(fn func(scoreboard.PlayerStats)) {
	sv.onKick = fn
	sv.redTable.setOnKick(fn)
	sv.bluTable.setOnKick(fn)
}

// Update renders the scoreboard snapshot, including rosters, scores, and elapsed time.
func (sv *ScoreboardView) Update(snap scoreboard.Snapshot) {
	sv.redTable.update(snap.Red, snap.Elapsed, snap.RedScore)
	sv.bluTable.update(snap.Blu, snap.Elapsed, snap.BluScore)
	sv.updateUnassigned(snap.Unassigned)
}

func (sv *ScoreboardView) updateUnassigned(players []scoreboard.PlayerStats) {
	objects := make([]fyne.CanvasObject, 0, len(players))
	for _, p := range players {
		lbl := widget.NewLabel(p.Name)
		objects = append(objects, lbl)
	}
	sv.unassigned.Objects = objects
	sv.unassigned.Refresh()
}

// Reset clears both tables and the unassigned list.
func (sv *ScoreboardView) Reset() {
	sv.redTable.update(nil, 0, 0)
	sv.bluTable.update(nil, 0, 0)
	sv.updateUnassigned(nil)
}

// View returns the scoreboard as a single canvas object.
func (sv *ScoreboardView) View() fyne.CanvasObject {
	return container.NewBorder(nil, sv.unassigned, nil, nil,
		container.NewHSplit(sv.bluTable.View(), sv.redTable.View()),
	)
}

// teamTable is a single sortable team scoreboard.
type teamTable struct {
	name      string
	teamColor color.Color
	align     fyne.TextAlign
	reverse   bool
	columns   []scoreboardColumn
	widths    []float32
	data      []scoreboard.PlayerStats
	elapsed   time.Duration
	sortCol   int
	sortAsc   bool
	onKick    func(scoreboard.PlayerStats)

	header             *fyne.Container
	content            *fyne.Container
	scroll             *container.Scroll
	rows               []*fyne.Container
	titleBar           fyne.CanvasObject
	titleLabel         *canvas.Text
	score              int
	refreshHeaderCells func()
}

func newTeamTable(name string, teamColor color.Color, align fyne.TextAlign, reverse bool) *teamTable {
	table := &teamTable{
		name:      name,
		teamColor: teamColor,
		align:     align,
		reverse:   reverse,
		sortAsc:   false,
	}

	table.columns = append([]scoreboardColumn(nil), scoreboardColumns...)
	table.widths = make([]float32, len(scoreboardColumns))
	for i, c := range scoreboardColumns {
		table.widths[i] = c.Width
	}
	if reverse {
		for i, j := 0, len(table.columns)-1; i < j; i, j = i+1, j-1 {
			table.columns[i], table.columns[j] = table.columns[j], table.columns[i]
			table.widths[i], table.widths[j] = table.widths[j], table.widths[i]
		}
	}

	for i, c := range table.columns {
		if c.Title == colKills {
			table.sortCol = i
			break
		}
	}

	table.titleBar, table.titleLabel = makeTitleBar(name, teamColor)
	table.header = makeHeader(table)
	table.content = container.New(&fullWidthVBox{gap: scoreboardRowGap})
	table.scroll = container.NewVScroll(table.content)
	table.scroll.SetMinSize(fyne.NewSize(rowTotalWidth(table.widths, scoreboardCellGap), 0))
	table.setScore(0)

	return table
}

func (table *teamTable) View() fyne.CanvasObject {
	return container.NewBorder(table.titleBar, nil, nil, nil,
		container.NewBorder(table.header, nil, nil, nil, table.scroll),
	)
}

func (table *teamTable) update(players []scoreboard.PlayerStats, elapsed time.Duration, score int) {
	table.data = append([]scoreboard.PlayerStats(nil), players...)
	table.elapsed = elapsed
	table.setScore(score)
	table.sort()
	table.rebuild()
}

func (table *teamTable) setScore(score int) {
	table.score = score
	if table.titleLabel == nil {
		return
	}
	table.titleLabel.Text = fmt.Sprintf("%s : %d", table.name, score)
	table.titleLabel.Refresh()
}

func (table *teamTable) sort() {
	if table.sortCol < 0 || table.sortCol >= len(table.columns) {
		table.sortCol = 0
		table.sortAsc = false
	}
	col := table.columns[table.sortCol]

	less := func(left, other scoreboard.PlayerStats) int {
		if !table.sortAsc {
			left, other = other, left
		}

		var primary int
		if col.Less != nil {
			if col.Less(left, other, table.elapsed) {
				primary = -1
			} else if col.Less(other, left, table.elapsed) {
				primary = 1
			}
		} else {
			primary = cmp.Compare(left.Name, other.Name)
		}
		if primary != 0 {
			return primary
		}

		// Tie-break on player name so the order stays deterministic.
		return cmp.Compare(left.Name, other.Name)
	}

	slices.SortStableFunc(table.data, less)
}

func (table *teamTable) setOnKick(fn func(scoreboard.PlayerStats)) {
	table.onKick = fn
	table.rebuild()
}

func (table *teamTable) setSort(col int) {
	if table.sortCol == col {
		table.sortAsc = !table.sortAsc
	} else {
		table.sortCol = col
		table.sortAsc = true
	}
	table.sort()
	if table.refreshHeaderCells != nil {
		table.refreshHeaderCells()
	}
	table.rebuild()
}

func (table *teamTable) rebuild() {
	// Reuse existing row containers so interactive widgets (kick buttons) are
	// not destroyed and recreated on every log event, which breaks hover/click.
	for len(table.rows) < len(table.data) {
		table.rows = append(table.rows, makeTeamRow(table.columns, table.widths, table.teamColor, table.reverse))
	}
	if len(table.rows) > len(table.data) {
		table.rows = table.rows[:len(table.data)]
	}

	objects := make([]fyne.CanvasObject, len(table.rows))
	for i, row := range table.rows {
		table.updateRow(i, row)
		objects[i] = row
	}
	table.content.Objects = objects
	table.content.Refresh()
}

func makeTitleBar(name string, c color.Color) (fyne.CanvasObject, *canvas.Text) {
	background := canvas.NewRectangle(c)
	background.FillColor = c
	background.StrokeColor = c
	background.StrokeWidth = 0
	lbl := canvas.NewText(name, color.White)
	lbl.TextStyle = fyne.TextStyle{Bold: true}
	lbl.TextSize = 16
	bar := container.NewStack(background, container.NewCenter(lbl))
	bar.Resize(fyne.NewSize(titleBarMinWidth, titleBarHeight))
	return bar, lbl
}

func makeHeader(table *teamTable) *fyne.Container {
	cells := make([]fyne.CanvasObject, len(table.columns))
	for idx, col := range table.columns {
		btn := widget.NewButton(col.Title, func() {}) // placeholder OnTapped set below
		btn.Importance = widget.LowImportance
		cells[idx] = btn
	}

	table.refreshHeaderCells = func() {
		for idx, col := range table.columns {
			btn, ok := cells[idx].(*widget.Button)
			if !ok {
				continue
			}
			text := col.Title
			if idx == table.sortCol {
				if table.sortAsc {
					text += " ▲"
				} else {
					text += " ▼"
				}
			}
			btn.SetText(text)
			colIdx := idx
			btn.OnTapped = func() { table.setSort(colIdx) }
		}
	}
	table.refreshHeaderCells()

	return container.New(&scoreboardRowLayout{widths: table.widths, gap: scoreboardCellGap, rightAlign: table.reverse}, cells...)
}

func (table *teamTable) updateRow(id widget.ListItemID, o fyne.CanvasObject) {
	row, ok := o.(*fyne.Container)
	if !ok {
		return
	}
	player := table.data[id]
	for i, col := range table.columns {
		cell := row.Objects[i]
		switch col.Title {
		case colClass:
			stack, okStack := cell.(*fyne.Container)
			if !okStack {
				continue
			}
			img, okImg := stack.Objects[1].(*canvas.Image)
			if !okImg {
				continue
			}
			img.Resource = assets.ClassIcon(player.Class)
			img.Refresh()
		case colKick:
			btn, okBtn := cell.(*widget.Button)
			if !okBtn {
				continue
			}
			p := player
			btn.OnTapped = func() {
				if table.onKick != nil {
					table.onKick(p)
				}
			}
		default:
			stack, okStack := cell.(*fyne.Container)
			if !okStack {
				continue
			}
			lbl, okLabel := stack.Objects[1].(*widget.Label)
			if !okLabel {
				continue
			}
			lbl.SetText(col.Fmt(player, table.elapsed))
			lbl.Alignment = table.align
		}
	}
}

func makeTeamRow(columns []scoreboardColumn, widths []float32, teamColor color.Color, reverse bool) *fyne.Container {
	cells := make([]fyne.CanvasObject, len(columns))
	for idx, col := range columns {
		if col.Title == colClass {
			background := canvas.NewRectangle(teamColor)
			background.FillColor = teamColor
			img := canvas.NewImageFromResource(nil)
			img.FillMode = canvas.ImageFillContain
			cells[idx] = container.NewStack(background, img)
			continue
		}
		if col.Title == colKick {
			btn := widget.NewButton("Kick", nil)
			btn.Importance = widget.DangerImportance
			cells[idx] = btn
			continue
		}
		cells[idx] = newDatumCell(teamColor)
	}
	return container.New(&scoreboardRowLayout{widths: widths, gap: scoreboardCellGap, rightAlign: reverse}, cells...)
}

func newDatumCell(c color.Color) *fyne.Container {
	bg := canvas.NewRectangle(c)
	bg.FillColor = c
	lbl := widget.NewLabel("")
	lbl.Wrapping = fyne.TextWrapOff
	lbl.Truncation = fyne.TextTruncateEllipsis
	return container.NewStack(bg, lbl)
}

// scoreboardRowLayout lays out row cells at fixed widths with small gaps.
type scoreboardRowLayout struct {
	widths     []float32
	gap        float32
	rightAlign bool
}

func (l *scoreboardRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	total := l.totalWidth()
	posX := float32(0)
	if l.rightAlign && size.Width >= total {
		posX = size.Width - total
	}
	for idx, obj := range objects {
		w := l.widths[idx]
		obj.Move(fyne.NewPos(posX, 0))
		obj.Resize(fyne.NewSize(w, size.Height))
		posX += w
		if idx < len(objects)-1 {
			posX += l.gap
		}
	}
}

func (l *scoreboardRowLayout) totalWidth() float32 {
	return rowTotalWidth(l.widths, l.gap)
}

func rowTotalWidth(widths []float32, gap float32) float32 {
	if len(widths) == 0 {
		return 0
	}
	total := float32(0)
	for _, w := range widths {
		total += w
	}
	return total + gap*float32(len(widths)-1)
}

// fullWidthVBox is a vertical layout that stretches every child to the full
// container width. This avoids the centering behavior of Fyne's normal VBox so
// right-aligned content inside children stays pinned to the right edge.
type fullWidthVBox struct {
	gap float32
}

func (l *fullWidthVBox) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	posY := float32(0)
	for idx, obj := range objects {
		h := obj.MinSize().Height
		obj.Move(fyne.NewPos(0, posY))
		obj.Resize(fyne.NewSize(size.Width, h))
		posY += h
		if idx < len(objects)-1 {
			posY += l.gap
		}
	}
}

func (l *fullWidthVBox) MinSize(objects []fyne.CanvasObject) fyne.Size {
	maxW := float32(0)
	totalH := float32(0)
	for idx, obj := range objects {
		s := obj.MinSize()
		if s.Width > maxW {
			maxW = s.Width
		}
		totalH += s.Height
		if idx < len(objects)-1 {
			totalH += l.gap
		}
	}
	return fyne.NewSize(maxW, totalH)
}

func (l *scoreboardRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	maxH := float32(0)
	for _, obj := range objects {
		if h := obj.MinSize().Height; h > maxH {
			maxH = h
		}
	}
	return fyne.NewSize(l.totalWidth(), maxH)
}

func classOrder(c logparse.PlayerClass) int {
	switch c {
	case logparse.ClassScout:
		return 0
	case logparse.ClassSoldier:
		return 1
	case logparse.ClassPyro:
		return 2
	case logparse.ClassDemoman:
		return 3
	case logparse.ClassHeavy:
		return 4
	case logparse.ClassEngineer:
		return 5
	case logparse.ClassMedic:
		return 6
	case logparse.ClassSniper:
		return 7
	case logparse.ClassSpy:
		return 8
	case logparse.ClassSpectator:
		return 9
	case logparse.ClassUnknown:
		return classOrderUnknown
	default:
		return classOrderUnknown
	}
}
