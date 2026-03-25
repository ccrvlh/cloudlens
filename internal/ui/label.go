package ui

import (
	"fmt"

	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
)

// Label is a read-only display element styled to match DropDown in Info panels.
type Label struct {
	*tview.TextView
	label string
	value string
}

// NewLabel creates a styled read-only label/value pair.
func NewLabel(label, value string) *Label {
	l := &Label{
		TextView: tview.NewTextView(),
		label:    label,
		value:    value,
	}
	l.SetDynamicColors(true)
	l.SetBackgroundColor(tcell.ColorDefault)
	l.SetBorderPadding(0, 0, 0, 0)
	l.render()
	return l
}

func (l *Label) render() {
	l.SetText(fmt.Sprintf("[orange::b]%s[-:-:-] [antiquewhite::]%s[-:-:-]", l.label, l.value))
}

// GetLabel returns the label string (used for alignment in Info).
func (l *Label) GetLabel() string {
	return l.label
}

// SetLabel updates the label and re-renders.
func (l *Label) SetLabel(label string) {
	l.label = label
	l.render()
}

// SetValue updates the value and re-renders.
func (l *Label) SetValue(value string) {
	l.value = value
	l.render()
}
