package render

import (
	"fmt"

	"github.com/derailed/tview"
	"github.com/one2nc/cloudlens/internal/aws"
)

type BedrockModel struct{}

func (b BedrockModel) Header() Header {
	return Header{
		HeaderColumn{Name: "Model-ID", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Provider", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Name", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Input", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Output", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Streaming", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
	}
}

func (b BedrockModel) Render(o interface{}, ns string, row *Row) error {
	resp, ok := o.(aws.BedrockModelResp)
	if !ok {
		return fmt.Errorf("expected BedrockModelResp, but got %T", o)
	}

	row.ID = ns
	row.Fields = Fields{
		resp.ModelId,
		resp.ProviderName,
		resp.ModelName,
		resp.InputModalities,
		resp.OutputModalities,
		resp.Streaming,
	}
	return nil
}
