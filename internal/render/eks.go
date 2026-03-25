package render

import (
	"fmt"

	"github.com/derailed/tview"
	"github.com/one2nc/cloudlens/internal/aws"
)

type EKSCluster struct{}

func (e EKSCluster) Header() Header {
	return Header{
		HeaderColumn{Name: "Name", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Status", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Version", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Created-At", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: true},
		HeaderColumn{Name: "Arn", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: true, MX: false, Time: false},
	}
}

func (e EKSCluster) Render(o interface{}, ns string, row *Row) error {
	resp, ok := o.(aws.EKSClusterResp)
	if !ok {
		return fmt.Errorf("expected EKSClusterResp, but got %T", o)
	}

	row.ID = ns
	row.Fields = Fields{
		resp.Name,
		resp.Status,
		resp.Version,
		resp.CreatedAt,
		resp.Arn,
	}
	return nil
}
