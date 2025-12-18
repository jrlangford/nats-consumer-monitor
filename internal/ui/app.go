package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/rivo/tview"

	"me-monitor/internal/monitor"
)

const (
	gridColumns   = 4
	flashDuration = 180 * time.Millisecond
)

// App encapsulates the terminal UI application.
type App struct {
	app    *tview.Application
	grid   *tview.Grid
	views  []*tview.TextView
	theme  Theme
	flashC *FlashController
}

// NewApp creates a new UI application.
func NewApp(numConsumers int) *App {
	theme := DefaultTheme()

	app := tview.NewApplication()

	// Calculate grid rows based on number of consumers
	rows := (numConsumers + gridColumns - 1) / gridColumns
	rowSizes := make([]int, rows)
	for i := range rowSizes {
		rowSizes[i] = 0 // 0 means equal distribution
	}

	grid := tview.NewGrid().
		SetRows(rowSizes...).
		SetColumns(0, 0, 0, 0)
	grid.SetBackgroundColor(theme.Background)

	views := make([]*tview.TextView, numConsumers)

	return &App{
		app:    app,
		grid:   grid,
		views:  views,
		theme:  theme,
		flashC: NewFlashController(),
	}
}

// SetupViews initializes the grid with views for each consumer.
func (a *App) SetupViews(consumers []monitor.ConsumerState) {
	for i, state := range consumers {
		tv := tview.NewTextView()
		tv.SetDynamicColors(true)
		tv.SetBackgroundColor(a.theme.Background)
		tv.SetBorder(true)
		tv.SetBorderColor(a.theme.Border)
		tv.SetTitle(fmt.Sprintf(" %s ", state.Ref.Consumer))

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
			a.updateViews(states)
		}
	}
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
	return fmt.Sprintf(
		"[yellow]Last Delivered:[-] Consumer seq: %s  Stream seq: %s  Last delivery: %s\n"+
			"[yellow]Ack Floor:[-]    Consumer seq: %s  Stream seq: %s  Last ack: %s\n"+
			"[yellow]Outstanding Acks:[-] %d of max %d\n"+
			"[yellow]Redelivered:[-] %d\n"+
			"[yellow]Unprocessed:[-] %d\n"+
			"[yellow]Waiting Pulls:[-] %d of max %d\n",
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
}

// Stop gracefully stops the application.
func (a *App) Stop() {
	a.app.Stop()
}
