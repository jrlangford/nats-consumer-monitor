package ui

import (
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var colorTagRegex = regexp.MustCompile(`\[[a-zA-Z0-9:#-]*\]`)

// stripColorTags removes tview color tags from text.
func stripColorTags(text string) string {
	return colorTagRegex.ReplaceAllString(text, "")
}

// SelectableTextView wraps a TextView with right-aligned title and double-click to copy.
type SelectableTextView struct {
	*tview.TextView
	fullTitle    string
	onTextCopied func(text string)
}

// NewSelectableTextView creates a new selectable text view.
func NewSelectableTextView() *SelectableTextView {
	tv := tview.NewTextView()
	stv := &SelectableTextView{
		TextView: tv,
	}
	return stv
}

// SetFullTitle sets the title that will be right-truncated to fit.
func (s *SelectableTextView) SetFullTitle(title string) *SelectableTextView {
	s.fullTitle = title
	return s
}

// SetTextCopiedFunc sets a callback for when text is copied.
func (s *SelectableTextView) SetTextCopiedFunc(fn func(text string)) *SelectableTextView {
	s.onTextCopied = fn
	return s
}

// Draw renders the view with a right-aligned (left-truncated) title.
func (s *SelectableTextView) Draw(screen tcell.Screen) {
	s.updateTitle()
	s.TextView.Draw(screen)
}

func (s *SelectableTextView) updateTitle() {
	if s.fullTitle == "" {
		return
	}

	// Get the rect including borders
	_, _, bw, _ := s.GetRect()
	availableWidth := bw - 4 // 2 for borders, 2 for title padding

	title := s.fullTitle
	if len(title) > availableWidth {
		if availableWidth > 3 {
			// Truncate from left, keeping the rightmost (most meaningful) part
			title = "…" + title[len(title)-availableWidth+1:]
		} else if availableWidth > 0 {
			title = title[len(title)-availableWidth:]
		} else {
			title = ""
		}
	}

	s.TextView.SetTitle(" " + title + " ")
}

// MouseHandler handles mouse events - double-click to copy entire content.
func (s *SelectableTextView) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		x, y := event.Position()
		bx, by, bw, bh := s.GetRect()

		// Check if click is within bounds
		if x < bx || x >= bx+bw || y < by || y >= by+bh {
			return false, nil
		}

		switch action {
		case tview.MouseLeftClick:
			setFocus(s)
			return true, nil

		case tview.MouseLeftDoubleClick:
			// Double-click copies entire content
			text := stripColorTags(s.GetText(true))
			if text != "" && s.onTextCopied != nil {
				s.onTextCopied(strings.TrimSpace(text))
			}
			return true, nil
		}

		return false, nil
	}
}
