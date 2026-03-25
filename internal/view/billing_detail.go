package view

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	awsV2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/one2nc/cloudlens/internal"
	"github.com/one2nc/cloudlens/internal/aws"
	"github.com/one2nc/cloudlens/internal/model"
)

const billingLoadingText = "  [darkgray::]loading…[-:-:-]\n"

type BillingDetail struct {
	*tview.Flex
	serviceName string
	app         *App
	cfg         awsV2.Config

	focusable []tview.Primitive
	focusIdx  int
}

func NewBillingDetail(app *App, serviceName string) *BillingDetail {
	return &BillingDetail{
		Flex:        tview.NewFlex(),
		serviceName: serviceName,
		app:         app,
	}
}

func (d *BillingDetail) Init(ctx context.Context) error {
	d.SetDirection(tview.FlexRow)
	d.SetBorder(true)
	d.SetBorderColor(tcell.ColorLightSkyBlue)
	d.SetTitle(fmt.Sprintf(" [aqua::b]Billing[-:-:-]  [lightskyblue::]%s[-:-:-] ", d.serviceName))
	d.SetInputCapture(d.keyboard)

	cfg, ok := ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		d.AddItem(billingErrPanel("Could not get AWS session"), 0, 1, true)
		return nil
	}
	d.cfg = cfg

	summaryP := billingPanel(" [aqua::b]▸ MONTHLY COSTS (last 6 months)[-:-:-] ", billingLoadingText)
	chartP := billingPanel(" [aqua::b]▸ TREND[-:-:-] ", billingLoadingText)

	d.focusable = []tview.Primitive{summaryP, chartP}
	for _, p := range d.focusable {
		p.(*tview.TextView).SetInputCapture(d.panelCapture(nil))
	}

	body := tview.NewFlex().SetDirection(tview.FlexRow)
	body.AddItem(summaryP, 0, 2, true)
	body.AddItem(chartP, 0, 1, false)
	d.AddItem(body, 0, 1, true)

	go func() {
		monthly, err := aws.GetServiceMonthlyCosts(d.cfg, d.serviceName, 6)
		d.app.QueueUpdateDraw(func() {
			if err != nil {
				summaryP.SetText(fmt.Sprintf("  [red::]Error: %v[-:-:-]\n", err))
				chartP.SetText("")
				return
			}
			summaryP.SetText(billingTableContent(monthly))
			chartP.SetText(billingBarChart(monthly))
		})
	}()

	return nil
}

func (d *BillingDetail) Start() {
	if len(d.focusable) > 0 {
		d.focusIdx = 0
		d.app.Application.SetFocus(d.focusable[0])
	}
}

func (d *BillingDetail) Stop() {}

func (d *BillingDetail) nextFocus() {
	if len(d.focusable) == 0 {
		return
	}
	d.focusIdx = (d.focusIdx + 1) % len(d.focusable)
	d.app.Application.SetFocus(d.focusable[d.focusIdx])
}

func (d *BillingDetail) prevFocus() {
	if len(d.focusable) == 0 {
		return
	}
	d.focusIdx = (d.focusIdx - 1 + len(d.focusable)) % len(d.focusable)
	d.app.Application.SetFocus(d.focusable[d.focusIdx])
}

func (d *BillingDetail) panelCapture(extra func(*tcell.EventKey) *tcell.EventKey) func(*tcell.EventKey) *tcell.EventKey {
	return func(evt *tcell.EventKey) *tcell.EventKey {
		switch evt.Key() {
		case tcell.KeyEscape:
			d.app.PrevCmd(evt)
			return nil
		case tcell.KeyTab:
			d.nextFocus()
			return nil
		case tcell.KeyBacktab:
			d.prevFocus()
			return nil
		}
		if extra != nil {
			return extra(evt)
		}
		return evt
	}
}

func (d *BillingDetail) keyboard(evt *tcell.EventKey) *tcell.EventKey {
	return d.panelCapture(nil)(evt)
}

func (d *BillingDetail) Name() string { return "BillingDetail" }

func (d *BillingDetail) Hints() model.MenuHints {
	return model.MenuHints{
		{Mnemonic: "tab", Description: "Next Panel", Visible: true},
		{Mnemonic: "esc", Description: "Back", Visible: true},
	}
}

func billingPanel(title, content string) *tview.TextView {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetBorder(true)
	tv.SetBorderColor(tcell.ColorSteelBlue)
	tv.SetBorderFocusColor(tcell.ColorDeepSkyBlue)
	tv.SetTitle(title)
	tv.SetTitleAlign(tview.AlignLeft)
	tv.SetTextColor(tcell.ColorWhite)
	tv.SetText(content)
	tv.SetScrollable(true)
	tv.SetWrap(false)
	tv.SetBackgroundColor(tcell.ColorDefault)
	return tv
}

func billingErrPanel(msg string) *tview.TextView {
	return billingPanel(" [red::]Error[-:-:-] ", "  [red::]"+msg+"[-:-:-]\n")
}

func billingTableContent(monthly []aws.BillingMonthlyResp) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  [lightskyblue::]%-12s  %12s  %s[-:-:-]\n", "Month", "Cost", "Unit")
	fmt.Fprintf(&b, "  [darkgray::]%s[-:-:-]\n", strings.Repeat("─", 34))
	total := 0.0
	unit := "USD"
	for _, m := range monthly {
		f, _ := strconv.ParseFloat(m.Amount, 64)
		total += f
		unit = m.Unit
		fmt.Fprintf(&b, "  [white::]%-12s[-:-:-]  [yellow::]%12.4f[-:-:-]  [darkgray::]%s[-:-:-]\n",
			m.Month, f, m.Unit)
	}
	fmt.Fprintf(&b, "  [darkgray::]%s[-:-:-]\n", strings.Repeat("─", 34))
	fmt.Fprintf(&b, "  [lightskyblue::]%-12s[-:-:-]  [aqua::b]%12.4f[-:-:-]  [darkgray::]%s[-:-:-]\n",
		"Total", total, unit)
	return b.String()
}

func billingBarChart(monthly []aws.BillingMonthlyResp) string {
	if len(monthly) == 0 {
		return "  [darkgray::]no data[-:-:-]\n"
	}

	maxVal := 0.0
	for _, m := range monthly {
		if f, err := strconv.ParseFloat(m.Amount, 64); err == nil && f > maxVal {
			maxVal = f
		}
	}

	const barWidth = 30
	var b strings.Builder
	for _, m := range monthly {
		f, _ := strconv.ParseFloat(m.Amount, 64)
		bars := 0
		if maxVal > 0 {
			bars = int(f / maxVal * float64(barWidth))
		}
		bar := strings.Repeat("█", bars)
		fmt.Fprintf(&b, "  [white::]%-10s[-:-:-]  [green::]%-*s[-:-:-]  [yellow::]%.4f[-:-:-]\n",
			m.Month, barWidth, bar, f)
	}
	return b.String()
}
