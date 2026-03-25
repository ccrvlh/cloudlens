package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/one2nc/cloudlens/internal/ui"
)

type APIGW struct {
	ResourceViewer
}

func NewAPIGW(resource string) ResourceViewer {
	var a APIGW
	a.ResourceViewer = NewBrowser(resource)
	a.AddBindKeysFn(a.bindKeys)
	return &a
}

func (a *APIGW) bindKeys(aa ui.KeyActions) {
	aa.Add(ui.KeyActions{
		ui.KeyShiftN:    ui.NewKeyAction("Sort Name", a.GetTable().SortColCmd("Name", true), true),
		ui.KeyShiftI:    ui.NewKeyAction("Sort ID", a.GetTable().SortColCmd("ID", true), true),
		ui.KeyShiftE:    ui.NewKeyAction("Sort Endpoint-Type", a.GetTable().SortColCmd("Endpoint-Type", true), true),
		tcell.KeyEscape: ui.NewKeyAction("Back", a.App().PrevCmd, false),
		tcell.KeyEnter:  ui.NewKeyAction("View", a.enterCmd, false),
	})
}

func (a *APIGW) enterCmd(evt *tcell.EventKey) *tcell.EventKey {
	apiId := a.GetTable().GetSelectedItem()
	if apiId != "" {
		detailView := NewAPIGWDetail(a.App(), apiId)
		a.App().inject(detailView)
	}
	return nil
}
