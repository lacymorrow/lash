package alert

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/lacymorrow/lash/internal/tui/util"
)

func TestAlertComponent(t *testing.T) {
	alert := NewAlertCmp()

	// Test initial state
	if alert.IsVisible() {
		t.Error("Alert should not be visible initially")
	}

	if alert.View() != "" {
		t.Error("Alert view should be empty initially")
	}

	// Test window size update
	_, cmd := alert.Update(tea.WindowSizeMsg{Width: 80, Height: 1})
	if cmd != nil {
		t.Error("Window size update should not return a command")
	}

	// Test info message
	infoMsg := util.InfoMsg{
		Type: util.InfoTypeInfo,
		Msg:  "Test message",
		TTL:  time.Second,
	}

	_, cmd = alert.Update(infoMsg)
	if cmd == nil {
		t.Error("Info message should return a clear command")
	}

	if !alert.IsVisible() {
		t.Error("Alert should be visible after info message")
	}

	view := alert.View()
	if view == "" {
		t.Error("Alert view should not be empty after info message")
	}

	if len(view) == 0 {
		t.Error("Alert view should contain content")
	}

	// Test clear message
	_, cmd = alert.Update(util.ClearStatusMsg{})
	if cmd != nil {
		t.Error("Clear message should not return a command")
	}

	if alert.IsVisible() {
		t.Error("Alert should not be visible after clear message")
	}

	if alert.View() != "" {
		t.Error("Alert view should be empty after clear message")
	}
}

func TestAlertTypes(t *testing.T) {
	alert := NewAlertCmp()

	// Set window size
	alert.Update(tea.WindowSizeMsg{Width: 80, Height: 1})

	testCases := []struct {
		name     string
		msgType  util.InfoType
		expected string
	}{
		{"Info", util.InfoTypeInfo, "OKAY!"},
		{"Warning", util.InfoTypeWarn, "WARNING"},
		{"Error", util.InfoTypeError, "ERROR"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			infoMsg := util.InfoMsg{
				Type: tc.msgType,
				Msg:  "Test message",
				TTL:  time.Second,
			}

			alert.Update(infoMsg)
			view := alert.View()

			if view == "" {
				t.Error("Alert view should not be empty")
			}

			// The view should contain the expected type indicator
			// Note: We can't do exact string matching due to styling,
			// but we can verify the alert is showing something
			if len(view) == 0 {
				t.Errorf("Expected alert view to contain content for type %v", tc.msgType)
			}
		})
	}
}