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

	"lan-server-manager/internal/logger"
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

	autoScroll bool
	lastOffset float32
}

func newLogViewer() *LogViewer {
	lv := &LogViewer{
		clearButton: widget.NewButton("Clear", nil),
		downButton:  widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), nil),
		statusLabel: widget.NewLabel(""),
		logBox:      container.NewVBox(),
		autoScroll:  true,
	}
	lv.logScroll = container.NewScroll(lv.logBox)

	lv.clearButton.OnTapped = lv.clear
	lv.downButton.OnTapped = lv.scrollDown

	return lv
}

// View returns the logs card as a single canvas object.
func (lv *LogViewer) View() fyne.CanvasObject {
	return widget.NewCard("Chat", "", container.NewBorder(
		container.NewBorder(
			nil, nil,
			lv.statusLabel,
			container.NewHBox(lv.clearButton, lv.downButton),
			nil,
		),
		nil, nil, nil,
		lv.logScroll,
	))
}

// SetTarget stores the log tail target.
func (lv *LogViewer) SetTarget(target logs.Target) {
	lv.target = target
}

// Target returns the current log tail target.
func (lv *LogViewer) Target() logs.Target { return lv.target }

// Stop terminates any active log tail. It does not clear displayed messages.
func (lv *LogViewer) Stop() {
	logger.Infof("log viewer stopping tail for %q", lv.target.ContainerName)
	lv.mu.Lock()
	stream := lv.stream
	lv.stream = nil
	lv.clearCh = nil
	lv.mu.Unlock()

	if stream != nil {
		go stream.Stop()
	}
	if lv.done != nil {
		close(lv.done)
		lv.done = nil
	}
}

func (lv *LogViewer) start() error {
	if lv.target.ContainerName == "" {
		return fmt.Errorf("enter container name")
	}
	logger.Infof("log viewer connecting to logs for %q (ssh_host=%q)", lv.target.ContainerName, lv.target.SSHHost)

	stream, err := logs.Tail(lv.target)
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

	if lv.target.SSHHost != "" {
		lv.statusLabel.SetText("Watching (ssh " + lv.target.SSHHost + ")...")
		logger.Infof("log viewer watching %q via ssh %s", lv.target.ContainerName, lv.target.SSHHost)
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
		logger.Infof("log viewer route ended for %q", lv.target.ContainerName)
		lv.mu.Lock()
		if lv.stream == stream {
			lv.stream = nil
		}
		lv.mu.Unlock()
		go stream.Stop()
		fyne.Do(func() {
			lv.statusLabel.SetText("Stopped")
		})
	}()

	for {
		select {
		case line, ok := <-stream.Lines:
			if !ok {
				return
			}
			if !isServerLogLine(line) || !isChatLogLine(line) {
				continue
			}
			chat, err := parseChatMessage(line)
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
			logger.Errorf("log viewer tail error for %q: %v", lv.target.ContainerName, err)
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
		if lv.autoScroll {
			diff := math.Abs(float64(lv.logScroll.Offset.Y - lv.lastOffset))
			if diff > 2 {
				lv.autoScroll = false
			}
		}

		row := container.NewHBox(
			chatText(msg.Time+":", color.NRGBA{R: 255, G: 255, B: 255, A: 255}),
			chatText(msg.Name, msg.Color),
			chatText(": "+msg.Message, color.NRGBA{R: 255, G: 255, B: 255, A: 255}),
		)
		lv.logBox.Add(row)

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

func chatText(text string, c color.Color) *canvas.Text {
	t := canvas.NewText(text, c)
	t.TextSize = theme.Size(theme.SizeNameText)
	return t
}
