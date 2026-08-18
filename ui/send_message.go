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
	messagePanel := &SendMessage{
		entry:  widget.NewEntry(),
		button: widget.NewButton("Send", onSend),
	}
	messagePanel.entry.SetPlaceHolder("Type a message...")
	messagePanel.entry.OnSubmitted = func(string) {
		if !messagePanel.button.Disabled() {
			onSend()
		}
	}
	return messagePanel
}

// Clear resets the message input field.
func (s *SendMessage) Clear() { s.entry.SetText("") }

// Text returns the current message text.
func (s *SendMessage) Text() string { return s.entry.Text }

// SetEnabled enables or disables the send button.
func (s *SendMessage) SetEnabled(enabled bool) {
	setButtonsEnabled(enabled, s.button)
}

// View returns the message row as a single canvas object.
func (s *SendMessage) View() fyne.CanvasObject {
	return newActionRow(s.entry, s.button)
}
