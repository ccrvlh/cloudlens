package view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/one2nc/cloudlens/internal/ui"
)

type Bedrock struct {
	ResourceViewer
}

func NewBedrock(resource string) ResourceViewer {
	var b Bedrock
	b.ResourceViewer = NewBrowser(resource)
	b.AddBindKeysFn(b.bindKeys)
	return &b
}

func (b *Bedrock) bindKeys(aa ui.KeyActions) {
	aa.Add(ui.KeyActions{
		ui.KeyShiftP:    ui.NewKeyAction("Sort Provider", b.GetTable().SortColCmd("Provider", true), true),
		ui.KeyShiftN:    ui.NewKeyAction("Sort Name", b.GetTable().SortColCmd("Name", true), true),
		tcell.KeyEscape: ui.NewKeyAction("Back", b.App().PrevCmd, false),
		tcell.KeyEnter:  ui.NewKeyAction("View", b.enterCmd, false),
	})
}

func (b *Bedrock) enterCmd(evt *tcell.EventKey) *tcell.EventKey {
	modelId := b.GetTable().GetSelectedItem()
	if modelId != "" {
		detailView := NewBedrockDetail(b.App(), modelId)
		b.App().inject(detailView)
	}
	return nil
}
