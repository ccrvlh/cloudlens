package render

import (
	"fmt"

	"github.com/derailed/tview"
	"github.com/one2nc/cloudlens/internal/aws"
)

type APIGateway struct{}

func (a APIGateway) Header() Header {
	return Header{
		HeaderColumn{Name: "ID", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Name", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Endpoint-Type", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Created-Date", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: true},
		HeaderColumn{Name: "Description", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: true, MX: false, Time: false},
	}
}

func (a APIGateway) Render(o interface{}, ns string, row *Row) error {
	resp, ok := o.(aws.APIGatewayResp)
	if !ok {
		return fmt.Errorf("expected APIGatewayResp, but got %T", o)
	}

	row.ID = ns
	row.Fields = Fields{
		resp.ID,
		resp.Name,
		resp.EndpointType,
		resp.CreatedDate,
		resp.Description,
	}
	return nil
}
