package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/one2nc/cloudlens/internal/ui"
)

type Billing struct {
	ResourceViewer
}

func NewBilling(resource string) ResourceViewer {
	var b Billing
	b.ResourceViewer = NewBrowser(resource)
	b.AddBindKeysFn(b.bindKeys)
	return &b
}

func (b *Billing) bindKeys(aa ui.KeyActions) {
	aa.Add(ui.KeyActions{
		ui.KeyShiftN:    ui.NewKeyAction("Sort Service", b.GetTable().SortColCmd("Service", true), true),
		ui.KeyShiftC:    ui.NewKeyAction("Sort Current-Month", b.GetTable().SortColCmd("Current-Month", true), true),
		ui.KeyShiftP:    ui.NewKeyAction("Sort Previous-Month", b.GetTable().SortColCmd("Previous-Month", true), true),
		tcell.KeyEscape: ui.NewKeyAction("Back", b.App().PrevCmd, false),
		tcell.KeyEnter:  ui.NewKeyAction("View", b.enterCmd, false),
	})
}

func (b *Billing) enterCmd(evt *tcell.EventKey) *tcell.EventKey {
	serviceName := b.GetTable().GetSelectedItem()
	if serviceName != "" {
		detailView := NewBillingDetail(b.App(), serviceName)
		b.App().inject(detailView)
	}
	return nil
}
