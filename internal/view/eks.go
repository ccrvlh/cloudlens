package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/one2nc/cloudlens/internal/ui"
)

type EKS struct {
	ResourceViewer
}

func NewEKS(resource string) ResourceViewer {
	var e EKS
	e.ResourceViewer = NewBrowser(resource)
	e.AddBindKeysFn(e.bindKeys)
	return &e
}

func (e *EKS) bindKeys(aa ui.KeyActions) {
	aa.Add(ui.KeyActions{
		ui.KeyShiftN:    ui.NewKeyAction("Sort Name", e.GetTable().SortColCmd("Name", true), true),
		ui.KeyShiftS:    ui.NewKeyAction("Sort Status", e.GetTable().SortColCmd("Status", true), true),
		ui.KeyShiftV:    ui.NewKeyAction("Sort Version", e.GetTable().SortColCmd("Version", true), true),
		tcell.KeyEscape: ui.NewKeyAction("Back", e.App().PrevCmd, false),
		tcell.KeyEnter:  ui.NewKeyAction("View", e.enterCmd, false),
	})
}

func (e *EKS) enterCmd(evt *tcell.EventKey) *tcell.EventKey {
	clusterName := e.GetTable().GetSelectedItem()
	if clusterName != "" {
		detailView := NewEKSDetail(e.App(), clusterName)
		e.App().inject(detailView)
	}
	return nil
}
