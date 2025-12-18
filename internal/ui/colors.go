package ui

import "github.com/gdamore/tcell/v2"

// Theme defines the color scheme for the UI.
type Theme struct {
	Background  tcell.Color
	Border      tcell.Color
	Flash       tcell.Color
	Title       tcell.Color
	Text        tcell.Color
	ErrorText   tcell.Color
	WarningText tcell.Color
}

// DefaultTheme returns the default dark theme.
func DefaultTheme() Theme {
	return Theme{
		Background:  tcell.NewRGBColor(24, 24, 37),
		Border:      tcell.NewRGBColor(88, 91, 112),
		Flash:       tcell.NewRGBColor(80, 120, 180),
		Title:       tcell.NewRGBColor(137, 180, 250),
		Text:        tcell.ColorWhite,
		ErrorText:   tcell.ColorRed,
		WarningText: tcell.ColorYellow,
	}
}
