package ui

import (
	"encoding/json"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/config"
)

// savedServerTab stores one saved tab's connection details.
type savedServerTab struct {
	Address  string `json:"address"`
	Password string `json:"password"`
	Map      string `json:"map"`
}

// Manager owns the tabbed interface and persists open tabs.
type Manager struct {
	window     fyne.Window
	tabs       *ServerTabs
	panels     []*ServerPanel
	plusTab    *container.TabItem
	prefs      fyne.Preferences
	tabCounter int
}

// NewManager creates the tabbed UI and restores any previously saved tabs.
func NewManager(window fyne.Window, prefs fyne.Preferences, cfg config.Config) *Manager {
	appConfig = cfg
	m := &Manager{window: window, prefs: prefs}
	m.buildUI()
	return m
}

// Content returns the root canvas object for the window.
func (m *Manager) Content() fyne.CanvasObject {
	return m.tabs
}

func (m *Manager) buildUI() {
	m.tabs = NewServerTabs(nil, m.handleTabClose, m.addPanel)

	// The "+" tab sits after the rightmost active tab and opens a new server tab.
	m.plusTab = container.NewTabItem("＋", widget.NewLabel("Click + to add a server"))

	m.loadTabs()
}

// loadTabs restores saved tab state, opens servers defined in the config, or
// creates one empty tab if neither is available.
func (m *Manager) loadTabs() {
	if len(appConfig.Servers) > 0 {
		for _, preset := range appConfig.Servers {
			p := m.newPanel(m.nextTabTitle())
			p.connection.SetAddress(preset.Address)
			p.connection.SetPassword(preset.RCONPassword)
			m.panels = append(m.panels, p)
		}

		m.rebuildTabs()
		m.tabs.SelectIndex(0)
		for _, p := range m.panels {
			p.connect()
		}
		m.saveTabs()
		return
	}

	raw := m.prefs.String("tabs")
	if raw == "" {
		m.addPanel()
		return
	}

	var saved []savedServerTab
	if err := json.Unmarshal([]byte(raw), &saved); err != nil {
		m.addPanel()
		return
	}

	for _, t := range saved {
		p := m.newPanel(m.nextTabTitle())
		p.connection.SetAddress(t.Address)
		p.connection.SetPassword(t.Password)
		p.actions.SetMap(t.Map)
		m.panels = append(m.panels, p)
	}

	if len(m.panels) == 0 {
		m.addPanel()
		return
	}

	m.rebuildTabs()
	m.tabs.SelectIndex(0)
}

// saveTabs writes the current set of server tabs to preferences.
func (m *Manager) saveTabs() {
	saved := make([]savedServerTab, len(m.panels))
	for i, p := range m.panels {
		saved[i] = savedServerTab{
			Address:  p.connection.Address(),
			Password: p.connection.Password(),
			Map:      p.actions.SelectedMap(),
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
	m.rebuildTabs()
	m.tabs.SelectIndex(len(m.panels) - 1)
	m.saveTabs()
}

func (m *Manager) rebuildTabs() {
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

func (m *Manager) handleTabClose(item *container.TabItem) {
	if item == m.plusTab {
		return
	}
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

	m.tabs.Remove(item)

	target := closedIndex
	if target >= len(m.panels) {
		target = len(m.panels) - 1
	}
	if target < 0 {
		target = 0
	}
	m.tabs.SelectIndex(target)

	m.saveTabs()
}
