package ui

import (
	"fmt"
	"image/color"
	"sort"
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
	colClass = "CLS"
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

var scoreboardColumns = []scoreboardColumn{
	{
		Title: colName,
		Width: 170,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return p.Name },
		Less:  func(a, b scoreboard.PlayerStats, _ time.Duration) bool { return a.Name < b.Name },
	},
	{
		Title: colClass,
		Width: 40,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return "" },
		Less: func(a, b scoreboard.PlayerStats, _ time.Duration) bool {
			return classOrder(a.Class) < classOrder(b.Class)
		},
	},
	{
		Title: colKills,
		Width: 40,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return fmt.Sprintf("%d", p.Kills) },
		Less:  func(a, b scoreboard.PlayerStats, _ time.Duration) bool { return a.Kills < b.Kills },
	},
	{
		Title: "A",
		Width: 40,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return fmt.Sprintf("%d", p.Assists) },
		Less:  func(a, b scoreboard.PlayerStats, _ time.Duration) bool { return a.Assists < b.Assists },
	},
	{
		Title: "D",
		Width: 40,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return fmt.Sprintf("%d", p.Deaths) },
		Less:  func(a, b scoreboard.PlayerStats, _ time.Duration) bool { return a.Deaths < b.Deaths },
	},
	{
		Title: "DMG",
		Width: 65,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return fmt.Sprintf("%d", p.Damage) },
		Less:  func(a, b scoreboard.PlayerStats, _ time.Duration) bool { return a.Damage < b.Damage },
	},
	{
		Title: "DPM",
		Width: 55,
		Fmt: func(p scoreboard.PlayerStats, elapsed time.Duration) string {
			return fmt.Sprintf("%.0f", p.DPM(elapsed))
		},
		Less: func(a, b scoreboard.PlayerStats, elapsed time.Duration) bool { return a.DPM(elapsed) < b.DPM(elapsed) },
	},
	{
		Title: "DT",
		Width: 60,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return fmt.Sprintf("%d", p.DamageTaken) },
		Less:  func(a, b scoreboard.PlayerStats, _ time.Duration) bool { return a.DamageTaken < b.DamageTaken },
	},
	{
		Title: "DTM",
		Width: 55,
		Fmt: func(p scoreboard.PlayerStats, elapsed time.Duration) string {
			return fmt.Sprintf("%.0f", p.DTM(elapsed))
		},
		Less: func(a, b scoreboard.PlayerStats, elapsed time.Duration) bool { return a.DTM(elapsed) < b.DTM(elapsed) },
	},
	{
		Title: "Heals",
		Width: 55,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return fmt.Sprintf("%d", p.Heals) },
		Less:  func(a, b scoreboard.PlayerStats, _ time.Duration) bool { return a.Heals < b.Heals },
	},
	{
		Title: "Cap",
		Width: 35,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return fmt.Sprintf("%d", p.Caps) },
		Less:  func(a, b scoreboard.PlayerStats, _ time.Duration) bool { return a.Caps < b.Caps },
	},
	{
		Title: "KD",
		Width: 45,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return fmt.Sprintf("%.2f", p.KD()) },
		Less:  func(a, b scoreboard.PlayerStats, _ time.Duration) bool { return a.KD() < b.KD() },
	},
	{
		Title: "KAD",
		Width: 55,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return fmt.Sprintf("%.2f", p.KAD()) },
		Less:  func(a, b scoreboard.PlayerStats, _ time.Duration) bool { return a.KAD() < b.KAD() },
	},
	{
		Title: "Ping",
		Width: 45,
		Fmt:   func(p scoreboard.PlayerStats, _ time.Duration) string { return fmt.Sprintf("%d", p.Ping) },
		Less:  func(a, b scoreboard.PlayerStats, _ time.Duration) bool { return a.Ping < b.Ping },
	},
	{
		Title: colKick,
		Width: 55,
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

// Update replaces the displayed rosters, team scores, and unassigned players.
// The elapsed duration is used for DPM/DTM columns.
func (sv *ScoreboardView) Update(red, blu, unassigned []scoreboard.PlayerStats, elapsed time.Duration, redScore, bluScore int) {
	sv.redTable.update(red, elapsed, redScore)
	sv.bluTable.update(blu, elapsed, bluScore)
	sv.updateUnassigned(unassigned)
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
	t := &teamTable{
		name:      name,
		teamColor: teamColor,
		align:     align,
		reverse:   reverse,
		sortAsc:   false,
	}

	t.columns = append([]scoreboardColumn(nil), scoreboardColumns...)
	t.widths = make([]float32, len(scoreboardColumns))
	for i, c := range scoreboardColumns {
		t.widths[i] = c.Width
	}
	if reverse {
		for i, j := 0, len(t.columns)-1; i < j; i, j = i+1, j-1 {
			t.columns[i], t.columns[j] = t.columns[j], t.columns[i]
			t.widths[i], t.widths[j] = t.widths[j], t.widths[i]
		}
	}

	for i, c := range t.columns {
		if c.Title == colKills {
			t.sortCol = i
			break
		}
	}

	t.titleBar, t.titleLabel = makeTitleBar(name, teamColor)
	t.header = makeHeader(t)
	t.content = container.New(&fullWidthVBox{gap: 2})
	t.scroll = container.NewVScroll(t.content)
	t.scroll.SetMinSize(fyne.NewSize(rowTotalWidth(t.widths, scoreboardCellGap), 0))
	t.setScore(0)

	return t
}

func (t *teamTable) View() fyne.CanvasObject {
	return container.NewBorder(t.titleBar, nil, nil, nil,
		container.NewBorder(t.header, nil, nil, nil, t.scroll),
	)
}

func (t *teamTable) update(players []scoreboard.PlayerStats, elapsed time.Duration, score int) {
	t.data = append([]scoreboard.PlayerStats(nil), players...)
	t.elapsed = elapsed
	t.setScore(score)
	t.sort()
	t.rebuild()
}

func (t *teamTable) setScore(score int) {
	t.score = score
	if t.titleLabel == nil {
		return
	}
	t.titleLabel.Text = fmt.Sprintf("%s — %d", t.name, score)
	t.titleLabel.Refresh()
}

func (t *teamTable) sort() {
	if t.sortCol < 0 || t.sortCol >= len(t.columns) {
		t.sortCol = 0
		t.sortAsc = false
	}
	col := t.columns[t.sortCol]
	less := func(i, j int) bool {
		if !t.sortAsc {
			i, j = j, i
		}
		a, b := t.data[i], t.data[j]

		primary := a.Name < b.Name
		if col.Less != nil {
			primary = col.Less(a, b, t.elapsed)
		}
		if primary {
			return true
		}

		// Col.Less says a >= b. Check the strictly-less reverse comparison to
		// detect ties; ties are broken by name so the order stays deterministic.
		reverse := b.Name < a.Name
		if col.Less != nil {
			reverse = col.Less(b, a, t.elapsed)
		}
		if reverse {
			return false
		}
		return a.Name < b.Name
	}
	sort.SliceStable(t.data, less)
}

func (t *teamTable) setOnKick(fn func(scoreboard.PlayerStats)) {
	t.onKick = fn
	t.rebuild()
}

func (t *teamTable) setSort(col int) {
	if t.sortCol == col {
		t.sortAsc = !t.sortAsc
	} else {
		t.sortCol = col
		t.sortAsc = true
	}
	t.sort()
	if t.refreshHeaderCells != nil {
		t.refreshHeaderCells()
	}
	t.rebuild()
}

func (t *teamTable) rebuild() {
	// Reuse existing row containers so interactive widgets (kick buttons) are
	// not destroyed and recreated on every log event, which breaks hover/click.
	for len(t.rows) < len(t.data) {
		t.rows = append(t.rows, makeTeamRow(t.columns, t.widths, t.teamColor, t.reverse))
	}
	if len(t.rows) > len(t.data) {
		t.rows = t.rows[:len(t.data)]
	}

	objects := make([]fyne.CanvasObject, len(t.rows))
	for i, row := range t.rows {
		t.updateRow(widget.ListItemID(i), row)
		objects[i] = row
	}
	t.content.Objects = objects
	t.content.Refresh()
}

func makeTitleBar(name string, c color.Color) (fyne.CanvasObject, *canvas.Text) {
	bg := canvas.NewRectangle(c)
	bg.FillColor = c
	bg.StrokeColor = c
	bg.StrokeWidth = 0
	lbl := canvas.NewText(name, color.White)
	lbl.TextStyle = fyne.TextStyle{Bold: true}
	lbl.TextSize = 16
	bar := container.NewStack(bg, container.NewCenter(lbl))
	bar.Resize(fyne.NewSize(200, 28))
	return bar, lbl
}

func makeHeader(t *teamTable) *fyne.Container {
	cells := make([]fyne.CanvasObject, len(t.columns))
	for i, col := range t.columns {
		btn := widget.NewButton(col.Title, func() {}) // placeholder OnTapped set below
		btn.Importance = widget.LowImportance
		cells[i] = btn
	}

	t.refreshHeaderCells = func() {
		for i, col := range t.columns {
			btn := cells[i].(*widget.Button)
			text := col.Title
			if i == t.sortCol {
				if t.sortAsc {
					text += " ▲"
				} else {
					text += " ▼"
				}
			}
			btn.SetText(text)
			c := i
			btn.OnTapped = func() { t.setSort(c) }
		}
	}
	t.refreshHeaderCells()

	return container.New(&scoreboardRowLayout{widths: t.widths, gap: scoreboardCellGap, rightAlign: t.reverse}, cells...)
}

func (t *teamTable) updateRow(id widget.ListItemID, o fyne.CanvasObject) {
	row := o.(*fyne.Container)
	player := t.data[id]
	for i, col := range t.columns {
		cell := row.Objects[i]
		switch col.Title {
		case colClass:
			stack := cell.(*fyne.Container)
			img := stack.Objects[1].(*canvas.Image)
			img.Resource = assets.ClassIcon(player.Class)
			img.Refresh()
		case colKick:
			btn := cell.(*widget.Button)
			p := player
			btn.OnTapped = func() {
				if t.onKick != nil {
					t.onKick(p)
				}
			}
		default:
			stack := cell.(*fyne.Container)
			lbl := stack.Objects[1].(*widget.Label)
			lbl.SetText(col.Fmt(player, t.elapsed))
			lbl.Alignment = t.align
		}
	}
}

func makeTeamRow(columns []scoreboardColumn, widths []float32, teamColor color.Color, reverse bool) *fyne.Container {
	cells := make([]fyne.CanvasObject, len(columns))
	for i, col := range columns {
		if col.Title == colClass {
			bg := canvas.NewRectangle(teamColor)
			bg.FillColor = teamColor
			img := canvas.NewImageFromResource(nil)
			img.FillMode = canvas.ImageFillContain
			cells[i] = container.NewStack(bg, img)
			continue
		}
		if col.Title == colKick {
			btn := widget.NewButton("Kick", nil)
			btn.Importance = widget.DangerImportance
			cells[i] = btn
			continue
		}
		cells[i] = newDatumCell(teamColor)
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
	x := float32(0)
	if l.rightAlign && size.Width >= total {
		x = size.Width - total
	}
	for i, obj := range objects {
		w := l.widths[i]
		obj.Move(fyne.NewPos(x, 0))
		obj.Resize(fyne.NewSize(w, size.Height))
		x += w
		if i < len(objects)-1 {
			x += l.gap
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
	y := float32(0)
	for i, obj := range objects {
		h := obj.MinSize().Height
		obj.Move(fyne.NewPos(0, y))
		obj.Resize(fyne.NewSize(size.Width, h))
		y += h
		if i < len(objects)-1 {
			y += l.gap
		}
	}
}

func (l *fullWidthVBox) MinSize(objects []fyne.CanvasObject) fyne.Size {
	maxW := float32(0)
	totalH := float32(0)
	for i, obj := range objects {
		s := obj.MinSize()
		if s.Width > maxW {
			maxW = s.Width
		}
		totalH += s.Height
		if i < len(objects)-1 {
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
	default:
		return 99
	}
}
