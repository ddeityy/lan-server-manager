package ui

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/logs"
)

const maxLogLines = 1000

// Game log lines start with "L <date> - <time>: ".
const logPrefixLen = 25

func isGameLogLine(line string) bool {
	if len(line) < logPrefixLen {
		return false
	}
	return line[0] == 'L' &&
		line[1] == ' ' &&
		line[12] == ' ' &&
		line[13] == '-' &&
		line[14] == ' ' &&
		line[23] == ':' &&
		line[24] == ' '
}

// formatLogLine strips the leading "L <date> - " prefix and keeps the time
// followed by the log message.
func formatLogLine(line string) string {
	if len(line) < logPrefixLen {
		return line
	}
	return line[15:23] + ": " + line[logPrefixLen:]
}

// LogViewer tails docker logs for a local container and displays them.
type LogViewer struct {
	mu             sync.Mutex
	containerEntry *widget.Entry
	watchButton    *widget.Button
	clearButton    *widget.Button
	downButton     *widget.Button
	statusLabel    *widget.Label
	logScroll      *container.Scroll
	logBox         *widget.Label

	stream     *logs.Stream
	lines      []string
	in         chan string
	clearCh    chan struct{}
	done       chan struct{}
	autoScroll bool
	lastOffset float32
}

func newLogViewer() *LogViewer {
	lv := &LogViewer{
		containerEntry: widget.NewEntry(),
		watchButton:    widget.NewButton("Watch", nil),
		clearButton:    widget.NewButton("Clear", nil),
		downButton:     widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), nil),
		statusLabel:    widget.NewLabel(""),
		logBox:         widget.NewLabel(""),
		autoScroll:     true,
	}
	lv.containerEntry.SetPlaceHolder("container_name")
	lv.logBox.Wrapping = fyne.TextWrapBreak
	lv.logScroll = container.NewScroll(lv.logBox)

	lv.watchButton.OnTapped = lv.toggle
	lv.clearButton.OnTapped = lv.clear
	lv.downButton.OnTapped = lv.scrollDown

	return lv
}

// View returns the logs card as a single canvas object.
func (lv *LogViewer) View() fyne.CanvasObject {
	return widget.NewCard("Logs", "", container.NewBorder(
		container.NewBorder(
			nil, nil,
			widget.NewLabel("Container:"),
			container.NewHBox(lv.watchButton, lv.clearButton, lv.downButton, lv.statusLabel),
			container.New(&minWidthLayout{width: 240}, lv.containerEntry),
		),
		nil, nil, nil,
		lv.logScroll,
	))
}

// SetContainerName sets the container name input field.
func (lv *LogViewer) SetContainerName(name string) { lv.containerEntry.SetText(name) }

// ContainerName returns the trimmed container name input value.
func (lv *LogViewer) ContainerName() string { return strings.TrimSpace(lv.containerEntry.Text) }

// Stop terminates any active log tail.
func (lv *LogViewer) Stop() {
	lv.mu.Lock()
	stream := lv.stream
	lv.stream = nil
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

	if err := lv.start(); err != nil {
		lv.statusLabel.SetText("Error: " + err.Error())
	}
}

func (lv *LogViewer) start() error {
	name := lv.ContainerName()
	if name == "" {
		return fmt.Errorf("enter container name")
	}

	stream, err := logs.Tail(name)
	if err != nil {
		return err
	}

	in := make(chan string, maxLogLines)
	clearCh := make(chan struct{})
	done := make(chan struct{})

	lv.mu.Lock()
	lv.stream = stream
	lv.in = in
	lv.clearCh = clearCh
	lv.done = done
	lv.mu.Unlock()

	lv.watchButton.SetText("Stop")
	lv.statusLabel.SetText("Watching...")

	go lv.collect(in, clearCh, done)
	go lv.route(stream, in, done)

	return nil
}

func (lv *LogViewer) clear() {
	if lv.clearCh != nil {
		select {
		case lv.clearCh <- struct{}{}:
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

func (lv *LogViewer) route(stream *logs.Stream, in chan<- string, done <-chan struct{}) {
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
			if !isGameLogLine(line) {
				continue
			}
			select {
			case in <- formatLogLine(line):
			case <-done:
				return
			}
		case err, ok := <-stream.Errors:
			if !ok {
				continue
			}
			msg := err.Error()
			fyne.Do(func() { lv.statusLabel.SetText("Error: " + msg) })
		case <-done:
			return
		}
	}
}

func (lv *LogViewer) collect(in <-chan string, clearCh <-chan struct{}, done <-chan struct{}) {
	for {
		select {
		case line, ok := <-in:
			if !ok {
				return
			}
			lv.appendLine(line)
		case <-clearCh:
			fyne.Do(func() {
				lv.lines = nil
				lv.logBox.SetText("")
			})
		case <-done:
			return
		}
	}
}

func (lv *LogViewer) appendLine(line string) {
	fyne.Do(func() {
		// If the user has scrolled away from the bottom, pause auto-scroll.
		if lv.autoScroll {
			diff := math.Abs(float64(lv.logScroll.Offset.Y - lv.lastOffset))
			if diff > 2 {
				lv.autoScroll = false
			}
		}

		lv.lines = append(lv.lines, line)
		if len(lv.lines) > maxLogLines {
			lv.lines = lv.lines[len(lv.lines)-maxLogLines:]
		}
		lv.logBox.SetText(strings.Join(lv.lines, "\n"))

		if lv.autoScroll {
			lv.logScroll.ScrollToBottom()
			lv.lastOffset = lv.logScroll.Offset.Y
		}
	})
}
