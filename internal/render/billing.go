package render

import (
	"fmt"
	"strconv"

	"github.com/derailed/tview"
	"github.com/one2nc/cloudlens/internal/aws"
)

type Billing struct{}

func (b Billing) Header() Header {
	return Header{
		HeaderColumn{Name: "Service", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Current-Month", SortIndicatorIdx: 0, Align: tview.AlignRight, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Previous-Month", SortIndicatorIdx: 0, Align: tview.AlignRight, Hide: false, Wide: false, MX: false, Time: false},
		HeaderColumn{Name: "Unit", SortIndicatorIdx: 0, Align: tview.AlignLeft, Hide: false, Wide: false, MX: false, Time: false},
	}
}

func (b Billing) Render(o interface{}, ns string, row *Row) error {
	resp, ok := o.(aws.BillingServiceResp)
	if !ok {
		return fmt.Errorf("expected BillingServiceResp, but got %T", o)
	}

	row.ID = ns
	row.Fields = Fields{
		resp.ServiceName,
		formatCost(resp.CurrentMonthCost),
		formatCost(resp.PreviousMonthCost),
		resp.Unit,
	}
	return nil
}

// formatCost renders a cost string to 4 decimal places, falling back to the
// raw string if it cannot be parsed as a float.
func formatCost(s string) string {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return fmt.Sprintf("%.4f", f)
	}
	return s
}
