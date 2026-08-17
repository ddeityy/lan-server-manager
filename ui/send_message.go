package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// SendMessage holds the message entry and send button rendered under the chat log.
type SendMessage struct {
	entry  *widget.Entry
	button *widget.Button
}

func newSendMessage(onSend func()) *SendMessage {
	s := &SendMessage{
		entry:  widget.NewEntry(),
		button: widget.NewButton("Send", onSend),
	}
	s.entry.SetPlaceHolder("Type a message...")
	s.entry.OnSubmitted = func(string) {
		if !s.button.Disabled() {
			onSend()
		}
	}
	return s
}

// Clear resets the message input field.
func (s *SendMessage) Clear() { s.entry.SetText("") }

// Text returns the current message text.
func (s *SendMessage) Text() string { return s.entry.Text }

// SetEnabled enables or disables the send button.
func (s *SendMessage) SetEnabled(enabled bool) {
	if enabled {
		s.button.Enable()
		return
	}
	s.button.Disable()
}

// View returns the message row as a single canvas object.
func (s *SendMessage) View() fyne.CanvasObject {
	return newActionRow(s.entry, s.button)
}
