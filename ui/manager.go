package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/config"
	"lan-server-manager/internal/logger"
	"lan-server-manager/logs"
)

// Manager owns the tabbed interface and the server panels it contains.
type Manager struct {
	window     fyne.Window
	tabs       *ServerTabs
	panels     []*ServerPanel
	plusTab    *container.TabItem
	tabCounter int
}

// NewManager creates the tabbed UI and opens servers defined in the config,
// or one empty tab if no servers are configured.
func NewManager(window fyne.Window, cfg config.Config) *Manager {
	logger.Infof("Creating manager with %d configured servers", len(cfg.Servers))
	appConfig = cfg
	m := &Manager{window: window}
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

// loadTabs opens servers defined in the config, or creates one empty tab if
// no servers are configured.
func (m *Manager) loadTabs() {
	if len(appConfig.Servers) > 0 {
		logger.Infof("Loading %d server tabs from config", len(appConfig.Servers))
		for _, preset := range appConfig.Servers {
			p := m.newPanel(m.nextTabTitle())
			p.connection.SetAddress(preset.Address)
			p.connection.SetPassword(preset.RCONPassword)
			p.actions.SetServerPassword(preset.Password)
			p.logs.SetTarget(logs.Target{
				ContainerName: preset.ContainerName,
				SSHHost:       preset.SSHHost,
				SSHUser:       preset.SSHUser,
				SSHPassword:   preset.SSHPassword,
				SSHKeyPath:    preset.SSHKeyPath,
			})
			logger.Infof("Created panel for %s (container=%q ssh_host=%q)", preset.Address, preset.ContainerName, preset.SSHHost)
			m.panels = append(m.panels, p)
		}

		m.rebuildTabs()
		m.tabs.SelectIndex(0)
		for _, p := range m.panels {
			p.connect()
		}
		return
	}

	logger.Infof("No configured servers, creating empty panel")
	m.addPanel()
}

func (m *Manager) addPanel() {
	title := m.nextTabTitle()
	logger.Infof("Adding new panel %s", title)
	p := m.newPanel(title)
	m.panels = append(m.panels, p)
	m.rebuildTabs()
	m.tabs.SelectIndex(len(m.panels) - 1)
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
			logger.Infof("Closing panel %q", item.Text)
			p.logs.Stop()
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
}
