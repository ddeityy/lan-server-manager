package ui

import (
	"fmt"
	"image/color"
	"math"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/game/logparse"
	"lan-server-manager/logger"
	"lan-server-manager/logs"
)

const maxLogLines = 1000

// LogViewer tails docker logs for a container and displays chat messages.
type LogViewer struct {
	mu          sync.Mutex
	clearButton *widget.Button
	downButton  *widget.Button
	statusLabel *widget.Label
	logScroll   *container.Scroll
	logBox      *fyne.Container

	target  logs.Target
	stream  *logs.Stream
	in      chan chatMessage
	clearCh chan struct{}
	done    chan struct{}
	onEvent func(logparse.Event)

	autoScroll bool
	lastOffset float32
}

func newLogViewer() *LogViewer {
	viewer := &LogViewer{
		clearButton: widget.NewButton("Clear", nil),
		downButton:  widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), nil),
		statusLabel: widget.NewLabel(""),
		logBox:      container.NewVBox(),
		autoScroll:  true,
	}
	viewer.logScroll = container.NewScroll(viewer.logBox)

	viewer.clearButton.OnTapped = viewer.clear
	viewer.downButton.OnTapped = viewer.scrollDown

	return viewer
}

// View returns the logs card as a single canvas object.
func (viewer *LogViewer) View() fyne.CanvasObject {
	return widget.NewCard("Chat", "", container.NewBorder(
		container.NewBorder(
			nil, nil,
			viewer.statusLabel,
			container.NewHBox(viewer.clearButton, viewer.downButton),
			nil,
		),
		nil, nil, nil,
		viewer.logScroll,
	))
}

// SetTarget stores the log tail target.
func (viewer *LogViewer) SetTarget(target logs.Target) {
	viewer.target = target
}

// Target returns the current log tail target.
func (viewer *LogViewer) Target() logs.Target { return viewer.target }

// SetOnEvent registers a callback invoked for every parsed log event.
func (viewer *LogViewer) SetOnEvent(fn func(logparse.Event)) {
	viewer.onEvent = fn
}

// Stop terminates any active log tail. It does not clear displayed messages.
func (viewer *LogViewer) Stop() {
	logger.Infof("log viewer stopping tail for %q", viewer.target.ContainerName)
	viewer.mu.Lock()
	stream := viewer.stream
	viewer.stream = nil
	viewer.clearCh = nil
	viewer.mu.Unlock()

	if stream != nil {
		go stream.Stop()
	}
	if viewer.done != nil {
		close(viewer.done)
		viewer.done = nil
	}
}

func (viewer *LogViewer) start() error {
	if viewer.target.ContainerName == "" {
		return fmt.Errorf("enter container name")
	}
	logger.Infof("log viewer connecting to logs for %q (ssh_host=%q)", viewer.target.ContainerName, viewer.target.SSHHost)

	stream, err := logs.Tail(viewer.target)
	if err != nil {
		return fmt.Errorf("tail logs: %w", err)
	}

	msgCh := make(chan chatMessage, maxLogLines)
	clearCh := make(chan struct{}, 1)
	done := make(chan struct{})

	viewer.mu.Lock()
	viewer.stream = stream
	viewer.in = msgCh
	viewer.clearCh = clearCh
	viewer.done = done
	viewer.mu.Unlock()

	if viewer.target.SSHHost != "" {
		viewer.statusLabel.SetText("Watching (ssh " + viewer.target.SSHHost + ")...")
		logger.Infof("log viewer watching %q via ssh %s", viewer.target.ContainerName, viewer.target.SSHHost)
	} else {
		viewer.statusLabel.SetText("Watching...")
	}

	go viewer.collect(msgCh, clearCh, done)
	go viewer.route(stream, msgCh, done)

	return nil
}

func (viewer *LogViewer) clear() {
	viewer.mu.Lock()
	clearCh := viewer.clearCh
	viewer.mu.Unlock()
	if clearCh != nil {
		select {
		case clearCh <- struct{}{}:
		default:
		}
	}
}

func (viewer *LogViewer) scrollDown() {
	viewer.autoScroll = true
	fyne.Do(func() {
		viewer.logScroll.ScrollToBottom()
		viewer.lastOffset = viewer.logScroll.Offset.Y
	})
}

func (viewer *LogViewer) route(stream *logs.Stream, messages chan<- chatMessage, done <-chan struct{}) {
	defer func() {
		logger.Infof("log viewer route ended for %q", viewer.target.ContainerName)
		viewer.mu.Lock()
		if viewer.stream == stream {
			viewer.stream = nil
		}
		viewer.mu.Unlock()
		go stream.Stop()
		fyne.Do(func() {
			viewer.statusLabel.SetText("Stopped")
		})
	}()

	for {
		select {
		case line, ok := <-stream.Lines:
			if !ok {
				return
			}
			evt, ok := logparse.Parse(line)
			if !ok {
				continue
			}
			if viewer.onEvent != nil {
				viewer.onEvent(evt)
			}
			if evt.Type != logparse.EventChat && evt.Type != logparse.EventChatTeam {
				continue
			}
			chat, chatOK := chatFromLogEvent(evt)
			if !chatOK {
				continue
			}
			select {
			case messages <- chat:
			case <-done:
				return
			}
		case err, ok := <-stream.Errors:
			if !ok {
				continue
			}
			logger.Errorf("log viewer tail error for %q: %v", viewer.target.ContainerName, err)
			msg := err.Error()
			fyne.Do(func() { viewer.statusLabel.SetText("Error: " + msg) })
		case <-done:
			return
		}
	}
}

func (viewer *LogViewer) collect(in <-chan chatMessage, clearCh <-chan struct{}, done <-chan struct{}) {
	for {
		select {
		case msg, ok := <-in:
			if !ok {
				return
			}
			viewer.appendLine(msg)
		case <-clearCh:
			fyne.Do(func() {
				viewer.logBox.Objects = nil
				viewer.logBox.Refresh()
			})
		case <-done:
			return
		}
	}
}

func (viewer *LogViewer) appendLine(msg chatMessage) {
	fyne.Do(func() {
		if viewer.autoScroll {
			diff := math.Abs(float64(viewer.logScroll.Offset.Y - viewer.lastOffset))
			if diff > 2 {
				viewer.autoScroll = false
			}
		}

		row := container.NewHBox(
			chatText(msg.Time+":", color.NRGBA{R: 255, G: 255, B: 255, A: 255}),
			chatText(msg.Name, msg.Color),
			chatText(": "+msg.Message, color.NRGBA{R: 255, G: 255, B: 255, A: 255}),
		)
		viewer.logBox.Add(row)

		for len(viewer.logBox.Objects) > maxLogLines {
			viewer.logBox.Objects = viewer.logBox.Objects[1:]
		}
		viewer.logBox.Refresh()

		if viewer.autoScroll {
			viewer.logScroll.ScrollToBottom()
			viewer.lastOffset = viewer.logScroll.Offset.Y
		}
	})
}

func chatText(text string, c color.Color) *canvas.Text {
	t := canvas.NewText(text, c)
	t.TextSize = theme.Size(theme.SizeNameText)
	return t
}
