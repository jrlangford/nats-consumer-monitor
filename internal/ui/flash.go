package ui

import (
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// FlashController manages flash animations for views.
// It prevents overlapping flash animations on the same view.
type FlashController struct {
	mu       sync.Mutex
	flashing map[*SelectableTextView]bool
}

// NewFlashController creates a new flash controller.
func NewFlashController() *FlashController {
	return &FlashController{
		flashing: make(map[*SelectableTextView]bool),
	}
}

// Flash triggers a flash animation on the given view if one isn't already running.
func (fc *FlashController) Flash(app *tview.Application, tv *SelectableTextView, base, flash tcell.Color, duration time.Duration) {
	fc.mu.Lock()
	if fc.flashing[tv] {
		fc.mu.Unlock()
		return // Skip if already flashing
	}
	fc.flashing[tv] = true
	fc.mu.Unlock()

	app.QueueUpdateDraw(func() {
		tv.SetBackgroundColor(flash)
	})

	time.AfterFunc(duration, func() {
		app.QueueUpdateDraw(func() {
			tv.SetBackgroundColor(base)
		})

		fc.mu.Lock()
		delete(fc.flashing, tv)
		fc.mu.Unlock()
	})
}

// IsFlashing returns true if the given view is currently flashing.
func (fc *FlashController) IsFlashing(tv *SelectableTextView) bool {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.flashing[tv]
}
