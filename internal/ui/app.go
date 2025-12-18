package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"me-monitor/internal/monitor"
)

const (
	gridColumns   = 4
	flashDuration = 180 * time.Millisecond
)

// App encapsulates the terminal UI application.
type App struct {
	app        *tview.Application
	grid       *tview.Grid
	views      []*SelectableTextView
	theme      Theme
	flashC     *FlashController
	throughput *monitor.ThroughputTracker
	lastStates []monitor.ConsumerState
	statusBar  *tview.TextView
}

// NewApp creates a new UI application.
func NewApp(numConsumers int) *App {
	theme := DefaultTheme()

	app := tview.NewApplication()

	// Calculate grid rows based on number of consumers (add 1 for status bar)
	rows := (numConsumers + gridColumns - 1) / gridColumns
	rowSizes := make([]int, rows+1)
	for i := range rows {
		rowSizes[i] = 0 // 0 means equal distribution
	}
	rowSizes[rows] = 1 // Status bar row

	grid := tview.NewGrid().
		SetRows(rowSizes...).
		SetColumns(0, 0, 0, 0)
	grid.SetBackgroundColor(theme.Background)

	views := make([]*SelectableTextView, numConsumers)

	// Create status bar
	statusBar := tview.NewTextView()
	statusBar.SetDynamicColors(true)
	statusBar.SetBackgroundColor(theme.Background)
	statusBar.SetTextAlign(tview.AlignCenter)
	statusBar.SetText("[dim]'t' throughput | 'c' clear | 'q' quit | double-click to copy[-]")
	grid.AddItem(statusBar, rows, 0, 1, gridColumns, 0, 0, false)

	return &App{
		app:        app,
		grid:       grid,
		views:      views,
		theme:      theme,
		flashC:     NewFlashController(),
		throughput: monitor.NewThroughputTracker(),
		statusBar:  statusBar,
	}
}

// copyToClipboard copies text to the system clipboard.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return fmt.Errorf("unsupported platform")
	}
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	_, _ = pipe.Write([]byte(text))
	pipe.Close()
	return cmd.Wait()
}

// SetupViews initializes the grid with views for each consumer.
func (a *App) SetupViews(consumers []monitor.ConsumerState) {
	for i, state := range consumers {
		tv := NewSelectableTextView()
		tv.SetDynamicColors(true)
		tv.SetBackgroundColor(a.theme.Background)
		tv.SetBorder(true)
		tv.SetBorderColor(a.theme.Border)
		tv.SetFullTitle(state.Ref.Consumer)
		tv.SetTextCopiedFunc(func(text string) {
			_ = copyToClipboard(text)
		})

		a.views[i] = tv

		row := i / gridColumns
		col := i % gridColumns

		a.grid.AddItem(tv, row, col, 1, 1, 0, 0, false)
	}
}

// Run starts the UI event loop.
func (a *App) Run(ctx context.Context, updates <-chan []monitor.ConsumerState) error {
	// Handle updates from poller
	go a.handleUpdates(ctx, updates)

	// Set up keyboard handler
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 't', 'T':
			a.toggleThroughput()
			return nil
		case 'c', 'C':
			a.clearThroughput()
			return nil
		case 'q', 'Q':
			a.app.Stop()
			return nil
		}
		return event
	})

	return a.app.SetRoot(a.grid, true).EnableMouse(true).Run()
}

func (a *App) handleUpdates(ctx context.Context, updates <-chan []monitor.ConsumerState) {
	firstUpdate := true

	for {
		select {
		case <-ctx.Done():
			a.app.Stop()
			return
		case states := <-updates:
			if firstUpdate {
				a.SetupViews(states)
				firstUpdate = false
			}
			a.lastStates = states
			a.throughput.Update(states)
			a.updateViews(states)
			a.updateStatusBar()
		}
	}
}

func (a *App) toggleThroughput() {
	if a.lastStates == nil {
		return
	}
	measuring := a.throughput.Toggle(a.lastStates)
	// Update status bar directly (we're on the main thread from input handler)
	if measuring {
		a.statusBar.SetText("[green]▶ Measuring...[-] 't' to stop")
	} else {
		a.statusBar.SetText("[yellow]■ Done[-] 't' restart | 'c' clear | double-click to copy")
	}
	// Views will be updated on next poll cycle
}

func (a *App) clearThroughput() {
	a.throughput.Clear()
	// Update status bar directly (we're on the main thread from input handler)
	a.statusBar.SetText("[dim]'t' throughput | 'c' clear | 'q' quit | double-click to copy[-]")
	// Views will be updated on next poll cycle
}

func (a *App) updateStatusBar() {
	if a.throughput.IsMeasuring() {
		return // Don't update while measuring - status is set by toggleThroughput
	}

	// Check if we have measurement results to display
	hasResults := false
	if a.lastStates != nil {
		for _, state := range a.lastStates {
			if m := a.throughput.Get(state.Ref.Stream, state.Ref.Consumer); m != nil {
				hasResults = true
				break
			}
		}
	}

	// This is called from handleUpdates goroutine, so use QueueUpdateDraw
	a.app.QueueUpdateDraw(func() {
		if hasResults {
			a.statusBar.SetText("[yellow]■ Done[-] 't' restart | 'c' clear | double-click to copy")
		} else {
			a.statusBar.SetText("[dim]'t' throughput | 'c' clear | 'q' quit | double-click to copy[-]")
		}
	})
}

func (a *App) updateViews(states []monitor.ConsumerState) {
	for i, state := range states {
		if i >= len(a.views) || a.views[i] == nil {
			continue
		}

		tv := a.views[i]
		text := a.formatConsumerState(state)
		shouldFlash := state.Changed && state.Error == nil

		a.app.QueueUpdateDraw(func() {
			tv.SetText(text)
			// Only reset background if not currently flashing
			if !a.flashC.IsFlashing(tv) {
				tv.SetBackgroundColor(a.theme.Background)
			}
		})

		if shouldFlash {
			a.flashC.Flash(a.app, tv, a.theme.Background, a.theme.Flash, flashDuration)
		}
	}
}

func (a *App) formatConsumerState(state monitor.ConsumerState) string {
	if state.Error != nil {
		return fmt.Sprintf("[red]ERROR[-]\n%v", state.Error)
	}

	ci := state.Info
	base := fmt.Sprintf(
		"[yellow]Last Delivered:[-] Consumer seq: %s  Stream seq: %s  Last delivery: %s\n"+
			"[yellow]Ack Floor:[-]    Consumer seq: %s  Stream seq: %s  Last ack: %s\n"+
			"[yellow]Outstanding Acks:[-] %d of max %d\n"+
			"[yellow]Redelivered:[-] %d\n"+
			"[yellow]Unprocessed:[-] %d\n"+
			"[yellow]Waiting Pulls:[-] %d of max %d",
		FormatInt(ci.Delivered.Consumer),
		FormatInt(ci.Delivered.Stream),
		Ago(ci.Delivered.Last),
		FormatInt(ci.AckFloor.Consumer),
		FormatInt(ci.AckFloor.Stream),
		Ago(ci.AckFloor.Last),
		ci.NumAckPending,
		ci.Config.MaxAckPending,
		ci.NumRedelivered,
		ci.NumPending,
		ci.NumWaiting,
		ci.Config.MaxWaiting,
	)

	// Add throughput info if available
	if m := a.throughput.Get(state.Ref.Stream, state.Ref.Consumer); m != nil {
		throughputInfo := fmt.Sprintf(
			"\n[cyan]─── Throughput ───[-]\n"+
				"[cyan]Duration:[-] %s\n"+
				"[cyan]Delivered:[-] %s msgs (%.1f/s)\n"+
				"[cyan]Acked:[-] %s msgs (%.1f/s)",
			m.Duration().Round(time.Second),
			FormatInt(m.DeliveredCount()),
			m.DeliveredRate(),
			FormatInt(m.AckedCount()),
			m.AckedRate(),
		)
		base += throughputInfo
	}

	return base
}

// Stop gracefully stops the application.
func (a *App) Stop() {
	a.app.Stop()
}
