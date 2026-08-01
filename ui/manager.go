package ui

import (
	"encoding/json"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// tabPrefs stores one saved tab's connection details.
type tabPrefs struct {
	Address  string `json:"address"`
	Password string `json:"password"`
	Map      string `json:"map"`
}

// Manager owns the tabbed interface and persists open tabs.
type Manager struct {
	window     fyne.Window
	tabs       *TabStrip
	panels     []*ServerPanel
	plusTab    *container.TabItem
	prefs      fyne.Preferences
	tabCounter int
}

// NewManager creates the tabbed UI and restores any previously saved tabs.
func NewManager(window fyne.Window, prefs fyne.Preferences) *Manager {
	m := &Manager{window: window, prefs: prefs}
	m.buildUI()
	return m
}

// Content returns the root canvas object for the window.
func (m *Manager) Content() fyne.CanvasObject {
	return m.tabs
}

func (m *Manager) buildUI() {
	docTabs := container.NewDocTabs()
	docTabs.CloseIntercept = m.onTabCloseIntercept
	docTabs.OnSelected = m.onTabSelected
	m.tabs = newTabStrip(docTabs)

	// The "+" tab sits after the rightmost active tab and opens a new server tab.
	// The full-width plus (U+FF0B) renders larger than a regular '+' in most fonts.
	m.plusTab = container.NewTabItem(" ＋ ", widget.NewLabel("Click + to add a server"))

	m.loadTabs()
}

// loadTabs restores saved tab state, or creates one empty tab if nothing is saved.
func (m *Manager) loadTabs() {
	raw := m.prefs.String("tabs")
	if raw == "" {
		m.addPanel()
		return
	}

	var saved []tabPrefs
	if err := json.Unmarshal([]byte(raw), &saved); err != nil {
		m.addPanel()
		return
	}

	for _, t := range saved {
		p := m.newPanel(m.nextTabTitle())
		p.addressEntry.SetText(t.Address)
		p.passwordEntry.SetText(t.Password)
		p.mapSelect.SetSelected(t.Map)
		m.panels = append(m.panels, p)
	}

	if len(m.panels) == 0 {
		m.addPanel()
		return
	}

	m.rebuildItems()
	m.tabs.SelectIndex(0)
	// The renderer may not exist yet; defer the first closability update.
	fyne.Do(m.updateTabClosability)
}

// saveTabs writes the current set of server tabs to preferences.
func (m *Manager) saveTabs() {
	saved := make([]tabPrefs, len(m.panels))
	for i, p := range m.panels {
		saved[i] = tabPrefs{
			Address:  p.addressEntry.Text,
			Password: p.passwordEntry.Text,
			Map:      p.mapSelect.Selected,
		}
	}

	raw, err := json.Marshal(saved)
	if err != nil {
		return
	}
	m.prefs.SetString("tabs", string(raw))
}

func (m *Manager) addPanel() {
	p := m.newPanel(m.nextTabTitle())
	m.panels = append(m.panels, p)
	m.rebuildItems()
	m.tabs.SelectIndex(len(m.panels) - 1)
	m.saveTabs()
	m.updateTabClosability()
}

func (m *Manager) rebuildItems() {
	items := make([]*container.TabItem, 0, len(m.panels)+1)
	for _, p := range m.panels {
		items = append(items, p.TabItem())
	}
	items = append(items, m.plusTab)
	m.tabs.SetItems(items)
}

func (m *Manager) newPanel(title string) *ServerPanel {
	return NewServerPanel(
		m.window,
		title,
		func() { fyne.Do(func() { m.tabs.Refresh() }) },
		m.saveTabs,
	)
}

func (m *Manager) nextTabTitle() string {
	m.tabCounter++
	return fmt.Sprintf("Server %d", m.tabCounter)
}

func (m *Manager) onTabSelected(item *container.TabItem) {
	if item == m.plusTab {
		m.addPanel()
	}
}

func (m *Manager) onTabCloseIntercept(item *container.TabItem) {
	if item == m.plusTab {
		return
	}
	// The last remaining server tab cannot be closed.
	if len(m.panels) <= 1 {
		return
	}
	m.closePanel(item)
}

func (m *Manager) closePanel(item *container.TabItem) {
	closedIndex := -1
	for i, p := range m.panels {
		if p.TabItem() == item {
			p.Disconnect()
			m.panels = append(m.panels[:i], m.panels[i+1:]...)
			closedIndex = i
			break
		}
	}

	if closedIndex == -1 {
		return
	}

	m.rebuildItems()
	m.saveTabs()

	target := closedIndex
	if target >= len(m.panels) {
		target = len(m.panels) - 1
	}
	if target < 0 {
		target = 0
	}
	m.tabs.SelectIndex(target)

	m.updateTabClosability()
}
