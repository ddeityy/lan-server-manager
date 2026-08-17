package ui

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/internal/logger"
	"lan-server-manager/logs"
)

const maxLogLines = 1000

// LogViewer tails docker logs for a local container and displays them.
type LogViewer struct {
	mu             sync.Mutex
	parser         *logParser
	containerEntry *widget.Entry
	watchButton    *widget.Button
	clearButton    *widget.Button
	downButton     *widget.Button
	statusLabel    *widget.Label
	logScroll      *container.Scroll
	logBox         *fyne.Container

	target     logs.Target
	stream     *logs.Stream
	in         chan chatMessage
	clearCh    chan struct{}
	done       chan struct{}
	autoScroll bool
	lastOffset float32
}

func newLogViewer() *LogViewer {
	lv := &LogViewer{
		parser:         newLogParser(),
		containerEntry: widget.NewEntry(),
		watchButton:    widget.NewButton("Watch", nil),
		clearButton:    widget.NewButton("Clear", nil),
		downButton:     widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), nil),
		statusLabel:    widget.NewLabel(""),
		logBox:         container.NewVBox(),
		autoScroll:     true,
	}
	lv.containerEntry.SetPlaceHolder("container_name")
	lv.logScroll = container.NewScroll(lv.logBox)

	lv.watchButton.OnTapped = lv.toggle
	lv.clearButton.OnTapped = lv.clear
	lv.downButton.OnTapped = lv.scrollDown

	return lv
}

// View returns the logs card as a single canvas object.
func (lv *LogViewer) View() fyne.CanvasObject {
	return widget.NewCard("Chat", "", container.NewBorder(
		container.NewVBox(
			container.NewBorder(
				nil, nil,
				widget.NewLabel("Container:"),
				container.NewHBox(lv.watchButton, lv.clearButton, lv.downButton),
				container.New(&minWidthLayout{width: 240}, lv.containerEntry),
			),
			lv.statusLabel,
		),
		nil, nil, nil,
		lv.logScroll,
	))
}

// SetTarget stores the log tail target and pre-fills the container name entry.
func (lv *LogViewer) SetTarget(target logs.Target) {
	lv.target = target
	lv.containerEntry.SetText(target.ContainerName)
}

// ContainerName returns the trimmed container name input value.
func (lv *LogViewer) ContainerName() string { return strings.TrimSpace(lv.containerEntry.Text) }

// Stop terminates any active log tail.
func (lv *LogViewer) Stop() {
	logger.Infof("log viewer stopping tail for %q", lv.ContainerName())
	lv.mu.Lock()
	stream := lv.stream
	lv.stream = nil
	lv.clearCh = nil
	lv.mu.Unlock()

	if stream != nil {
		stream.Stop()
	}
	if lv.done != nil {
		close(lv.done)
		lv.done = nil
	}
}

func (lv *LogViewer) toggle() {
	lv.mu.Lock()
	running := lv.stream != nil
	lv.mu.Unlock()

	if running {
		lv.Stop()
		fyne.Do(func() {
			lv.watchButton.SetText("Watch")
			lv.statusLabel.SetText("Stopped")
		})
		return
	}

	logger.Infof("log viewer starting tail for %q", lv.ContainerName())
	if err := lv.start(); err != nil {
		logger.Errorf("log viewer start failed: %v", err)
		lv.statusLabel.SetText("Error: " + err.Error())
	}
}

func (lv *LogViewer) start() error {
	target := lv.target
	if target.ContainerName == "" {
		target.ContainerName = lv.ContainerName()
	}
	if target.ContainerName == "" {
		return fmt.Errorf("enter container name")
	}
	logger.Infof("log viewer connecting to logs for %q (ssh_host=%q)", target.ContainerName, target.SSHHost)

	stream, err := logs.Tail(target)
	if err != nil {
		return err
	}

	in := make(chan chatMessage, maxLogLines)
	clearCh := make(chan struct{}, 1)
	done := make(chan struct{})

	lv.mu.Lock()
	lv.stream = stream
	lv.in = in
	lv.clearCh = clearCh
	lv.done = done
	lv.mu.Unlock()

	lv.watchButton.SetText("Stop")
	if target.SSHHost != "" {
		lv.statusLabel.SetText("Watching (ssh " + target.SSHHost + ")...")
		logger.Infof("log viewer watching %q via ssh %s", target.ContainerName, target.SSHHost)
	} else {
		lv.statusLabel.SetText("Watching...")
	}

	go lv.collect(in, clearCh, done)
	go lv.route(stream, in, done)

	return nil
}

func (lv *LogViewer) clear() {
	lv.mu.Lock()
	clearCh := lv.clearCh
	lv.mu.Unlock()
	if clearCh != nil {
		select {
		case clearCh <- struct{}{}:
		default:
		}
	}
}

func (lv *LogViewer) scrollDown() {
	lv.autoScroll = true
	fyne.Do(func() {
		lv.logScroll.ScrollToBottom()
		lv.lastOffset = lv.logScroll.Offset.Y
	})
}

func (lv *LogViewer) route(stream *logs.Stream, in chan<- chatMessage, done <-chan struct{}) {
	defer func() {
		lv.mu.Lock()
		if lv.stream == stream {
			lv.stream = nil
		}
		lv.mu.Unlock()
		stream.Stop()
		fyne.Do(func() {
			lv.watchButton.SetText("Watch")
			lv.statusLabel.SetText("Stopped")
		})
	}()

	for {
		select {
		case line, ok := <-stream.Lines:
			if !ok {
				return
			}
			if !lv.parser.isGameLogLine(line) || !lv.parser.isChatLogLine(line) {
				continue
			}
			chat, err := lv.parser.parseChat(line)
			if err != nil {
				continue
			}
			select {
			case in <- chat:
			case <-done:
				return
			}
		case err, ok := <-stream.Errors:
			if !ok {
				continue
			}
			logger.Errorf("log viewer tail error for %q: %v", lv.ContainerName(), err)
			msg := err.Error()
			fyne.Do(func() { lv.statusLabel.SetText("Error: " + msg) })
		case <-done:
			return
		}
	}
}

func (lv *LogViewer) collect(in <-chan chatMessage, clearCh <-chan struct{}, done <-chan struct{}) {
	for {
		select {
		case msg, ok := <-in:
			if !ok {
				return
			}
			lv.appendLine(msg)
		case <-clearCh:
			fyne.Do(func() {
				lv.logBox.Objects = nil
				lv.logBox.Refresh()
			})
		case <-done:
			return
		}
	}
}

func (lv *LogViewer) appendLine(msg chatMessage) {
	fyne.Do(func() {
		// If the user has scrolled away from the bottom, pause auto-scroll.
		if lv.autoScroll {
			diff := math.Abs(float64(lv.logScroll.Offset.Y - lv.lastOffset))
			if diff > 2 {
				lv.autoScroll = false
			}
		}

		lineText := canvas.NewText(msg.String(), chatColor(msg.Team))
		lineText.TextSize = theme.Size(theme.SizeNameText)
		lv.logBox.Add(lineText)

		for len(lv.logBox.Objects) > maxLogLines {
			lv.logBox.Objects = lv.logBox.Objects[1:]
		}
		lv.logBox.Refresh()

		if lv.autoScroll {
			lv.logScroll.ScrollToBottom()
			lv.lastOffset = lv.logScroll.Offset.Y
		}
	})
}

func chatColor(team string) color.Color {
	switch team {
	case "RED":
		return color.NRGBA{R: 167, G: 88, B: 75, A: 255}
	case "BLU":
		return color.NRGBA{R: 84, G: 125, B: 140, A: 255}
	case "CON":
		return color.NRGBA{R: 160, G: 160, B: 160, A: 255}
	case "SPC":
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	return theme.Color(theme.ColorNameForeground)
}
