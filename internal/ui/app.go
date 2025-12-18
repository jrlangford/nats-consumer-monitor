package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"me-monitor/internal/config"
	"me-monitor/internal/monitor"
)

const (
	flashDuration     = 180 * time.Millisecond
	windowPageFmt     = "window-%d"
	defaultStatusText = "[dim]'t' throughput | 'c' clear | '<'/'>' windows | 'q'/Ctrl-C quit | double-click to copy[-]"
)

// WindowPanel represents a single window/panel in the UI.
type WindowPanel struct {
	config     config.WindowConfig
	grid       *tview.Grid
	views      []*SelectableTextView
	viewMap    map[string]*SelectableTextView // keyed by "stream/consumer"
	statusBar  *tview.TextView
	throughput *monitor.ThroughputTracker
	theme      Theme
	flashC     *FlashController
}

// App encapsulates the terminal UI application.
type App struct {
	app        *tview.Application
	pages      *tview.Pages
	panels     []*WindowPanel
	theme      Theme
	currentIdx int
	lastStates []monitor.ConsumerState
}

// NewApp creates a new UI application with multiple window panels.
func NewApp(windows []config.WindowConfig) *App {
	theme := DefaultTheme()
	app := tview.NewApplication()

	panels := make([]*WindowPanel, len(windows))
	pages := tview.NewPages()

	for i, win := range windows {
		panel := newWindowPanel(win, theme)
		panels[i] = panel
		pages.AddPage(fmt.Sprintf(windowPageFmt, i), panel.grid, true, i == 0)
	}

	return &App{
		app:        app,
		pages:      pages,
		panels:     panels,
		theme:      theme,
		currentIdx: 0,
	}
}

func newWindowPanel(win config.WindowConfig, theme Theme) *WindowPanel {
	numConsumers := len(win.Consumers)
	columns := win.Columns
	if columns <= 0 {
		columns = 4
	}

	// Calculate grid rows based on number of consumers (add 1 for status bar)
	rows := (numConsumers + columns - 1) / columns
	rowSizes := make([]int, rows+1)
	for i := range rows {
		rowSizes[i] = 0 // 0 means equal distribution
	}
	rowSizes[rows] = 1 // Status bar row

	colSizes := make([]int, columns)
	for i := range colSizes {
		colSizes[i] = 0 // equal distribution
	}

	grid := tview.NewGrid().
		SetRows(rowSizes...).
		SetColumns(colSizes...)
	grid.SetBackgroundColor(theme.Background)

	views := make([]*SelectableTextView, numConsumers)
	viewMap := make(map[string]*SelectableTextView)

	// Create status bar
	statusBar := tview.NewTextView()
	statusBar.SetDynamicColors(true)
	statusBar.SetBackgroundColor(theme.Background)
	statusBar.SetTextAlign(tview.AlignCenter)
	statusBar.SetText(defaultStatusText)
	grid.AddItem(statusBar, rows, 0, 1, columns, 0, 0, false)

	return &WindowPanel{
		config:     win,
		grid:       grid,
		views:      views,
		viewMap:    viewMap,
		statusBar:  statusBar,
		throughput: monitor.NewThroughputTracker(),
		theme:      theme,
		flashC:     NewFlashController(),
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

// SetupViews initializes the grid with views for each consumer in a panel.
func (p *WindowPanel) SetupViews(allStates []monitor.ConsumerState) {
	// Build a map of all states for quick lookup
	stateMap := make(map[string]monitor.ConsumerState)
	for _, state := range allStates {
		key := state.Ref.Stream + "/" + state.Ref.Consumer
		stateMap[key] = state
	}

	columns := p.config.Columns
	for i, ref := range p.config.Consumers {
		tv := NewSelectableTextView()
		tv.SetDynamicColors(true)
		tv.SetBackgroundColor(p.theme.Background)
		tv.SetBorder(true)
		tv.SetBorderColor(p.theme.Border)
		tv.SetFullTitle(ref.Consumer)
		tv.SetTextCopiedFunc(func(text string) {
			_ = copyToClipboard(text)
		})

		p.views[i] = tv
		key := ref.Stream + "/" + ref.Consumer
		p.viewMap[key] = tv

		row := i / columns
		col := i % columns

		p.grid.AddItem(tv, row, col, 1, 1, 0, 0, false)
	}
}

// Run starts the UI event loop.
func (a *App) Run(ctx context.Context, updates <-chan []monitor.ConsumerState) error {
	// Handle updates from poller
	go a.handleUpdates(ctx, updates)

	// Set up keyboard handler
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle Ctrl-C
		if event.Key() == tcell.KeyCtrlC {
			a.app.Stop()
			return nil
		}

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
		case '<', ',':
			a.prevWindow()
			return nil
		case '>', '.':
			a.nextWindow()
			return nil
		}

		// Handle arrow keys for window navigation
		switch event.Key() {
		case tcell.KeyLeft:
			a.prevWindow()
			return nil
		case tcell.KeyRight:
			a.nextWindow()
			return nil
		}

		return event
	})

	return a.app.SetRoot(a.pages, true).EnableMouse(true).Run()
}

func (a *App) prevWindow() {
	if len(a.panels) <= 1 {
		return
	}
	a.currentIdx--
	if a.currentIdx < 0 {
		a.currentIdx = len(a.panels) - 1
	}
	a.pages.SwitchToPage(fmt.Sprintf(windowPageFmt, a.currentIdx))
	a.updateWindowTitle()
}

func (a *App) nextWindow() {
	if len(a.panels) <= 1 {
		return
	}
	a.currentIdx++
	if a.currentIdx >= len(a.panels) {
		a.currentIdx = 0
	}
	a.pages.SwitchToPage(fmt.Sprintf(windowPageFmt, a.currentIdx))
	a.updateWindowTitle()
}

func (a *App) updateWindowTitle() {
	if len(a.panels) <= 1 {
		return
	}
	panel := a.panels[a.currentIdx]
	title := fmt.Sprintf("[green]%s[-] (%d/%d)", panel.config.Name, a.currentIdx+1, len(a.panels))
	panel.statusBar.SetText(title + " [dim]| 't' throughput | 'c' clear | '<'/'>' windows | 'q'/Ctrl-C quit[-]")
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
				// Setup views for all panels
				for _, panel := range a.panels {
					panel.SetupViews(states)
				}
				firstUpdate = false
				a.updateWindowTitle()
			}
			a.lastStates = states
			// Update all panels with new states
			for _, panel := range a.panels {
				panel.throughput.Update(states)
				panel.updateViews(a.app, states)
				panel.updateStatusBar(a.app, states, a.currentIdx, len(a.panels))
			}
		}
	}
}

func (a *App) toggleThroughput() {
	if a.lastStates == nil {
		return
	}
	// Toggle throughput on all panels
	for _, panel := range a.panels {
		measuring := panel.throughput.Toggle(a.lastStates)
		// Update status bar directly (we're on the main thread from input handler)
		if measuring {
			panel.statusBar.SetText("[green]▶ Measuring...[-] 't' to stop")
		} else {
			panel.statusBar.SetText("[yellow]■ Done[-] 't' restart | 'c' clear | '<'/'>' windows | double-click to copy")
		}
	}
}

func (a *App) clearThroughput() {
	// Clear throughput on all panels
	for _, panel := range a.panels {
		panel.throughput.Clear()
		panel.statusBar.SetText(defaultStatusText)
	}
}

func (p *WindowPanel) updateStatusBar(app *tview.Application, states []monitor.ConsumerState, currentIdx, totalPanels int) {
	if p.throughput.IsMeasuring() {
		return // Don't update while measuring - status is set by toggleThroughput
	}

	// Check if we have measurement results to display
	hasResults := false
	for _, ref := range p.config.Consumers {
		if m := p.throughput.Get(ref.Stream, ref.Consumer); m != nil {
			hasResults = true
			break
		}
	}

	// This is called from handleUpdates goroutine, so use QueueUpdateDraw
	app.QueueUpdateDraw(func() {
		var prefix string
		if totalPanels > 1 {
			prefix = fmt.Sprintf("[green]%s[-] (%d/%d) | ", p.config.Name, currentIdx+1, totalPanels)
		}
		if hasResults {
			p.statusBar.SetText(prefix + "[yellow]■ Done[-] 't' restart | 'c' clear | '<'/'>' windows | double-click to copy")
		} else {
			p.statusBar.SetText(prefix + defaultStatusText)
		}
	})
}

func (p *WindowPanel) updateViews(app *tview.Application, states []monitor.ConsumerState) {
	// Build a map of states for quick lookup
	stateMap := make(map[string]monitor.ConsumerState)
	for _, state := range states {
		key := state.Ref.Stream + "/" + state.Ref.Consumer
		stateMap[key] = state
	}

	for _, ref := range p.config.Consumers {
		key := ref.Stream + "/" + ref.Consumer
		tv := p.viewMap[key]
		if tv == nil {
			continue
		}

		state, ok := stateMap[key]
		if !ok {
			continue
		}

		text := p.formatConsumerState(state)
		shouldFlash := state.Changed && state.Error == nil

		app.QueueUpdateDraw(func() {
			tv.SetText(text)
			// Only reset background if not currently flashing
			if !p.flashC.IsFlashing(tv) {
				tv.SetBackgroundColor(p.theme.Background)
			}
		})

		if shouldFlash {
			p.flashC.Flash(app, tv, p.theme.Background, p.theme.Flash, flashDuration)
		}
	}
}

func (p *WindowPanel) formatConsumerState(state monitor.ConsumerState) string {
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
	if m := p.throughput.Get(state.Ref.Stream, state.Ref.Consumer); m != nil {
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
