package ui

import (
	"fmt"
	"strings"

	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
)

const DropdownPadSpaces int = 3

type Info struct {
	*tview.Flex
	items map[string]tview.Primitive
}

func NewInfo(items map[string]tview.Primitive) *Info {
	i := Info{
		Flex:  tview.NewFlex(),
		items: items,
	}
	i.padDropDownLabels()
	i.build()
	return &i
}

func (i *Info) padDropDownLabels() {
	maxLabelLen := 0
	maxOptionLen := 0
	for _, p := range i.items {
		switch v := p.(type) {
		case *DropDown:
			// Use v.label (raw, no markup) for accurate length comparison
			if len(v.label) > maxLabelLen {
				maxLabelLen = len(v.label)
			}
			if v.GetFieldWidth() > maxOptionLen {
				maxOptionLen = v.GetFieldWidth()
			}
		case *Label:
			if len(v.label) > maxLabelLen {
				maxLabelLen = len(v.label)
			}
		}
	}

	for _, p := range i.items {
		switch v := p.(type) {
		case *DropDown:
			v.SetFieldWidth(maxOptionLen + DropdownPadSpaces)
			padding := strings.Repeat(" ", (maxLabelLen-len(v.label))+1)
			v.SetLabel(fmt.Sprintf("[orange::b]%s%s", v.label, padding))
		case *Label:
			padding := strings.Repeat(" ", (maxLabelLen-len(v.label))+1)
			v.SetLabel(v.label + padding)
		}
	}
}

func (i *Info) build() {
	i.Clear()
	i.SetDirection(tview.FlexRow)
	i.SetBorderColor(tcell.ColorBlack.TrueColor())
	i.SetBorderPadding(0, 4, 1, 1)
	for _, k := range SortMapKeys(i.items) {
		i.AddItem(i.items[k], 0, 1, false)
	}
}
