package view

import (
	"context"
	"fmt"
	"strings"
	"time"

	awsV2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/one2nc/cloudlens/internal"
	"github.com/one2nc/cloudlens/internal/aws"
	"github.com/one2nc/cloudlens/internal/model"
)

const bedrockLoadingText = "  [darkgray::]loading…[-:-:-]\n"

type BedrockDetail struct {
	*tview.Flex
	modelId string
	app     *App
	cfg     awsV2.Config
	detail  *aws.BedrockModelDetailResp

	focusable  []tview.Primitive
	focusIdx   int
	metricsPanel *tview.TextView
	cancelMon  context.CancelFunc

	jsonMode   bool
	prettyBody tview.Primitive
	jsonBody   tview.Primitive
}

func NewBedrockDetail(app *App, modelId string) *BedrockDetail {
	return &BedrockDetail{
		Flex:    tview.NewFlex(),
		modelId: modelId,
		app:     app,
	}
}

func (d *BedrockDetail) Init(ctx context.Context) error {
	d.SetDirection(tview.FlexRow)
	d.SetBorder(true)
	d.SetBorderColor(tcell.ColorLightSkyBlue)
	d.SetTitle(fmt.Sprintf(" [aqua::b]Bedrock[-:-:-]  [lightskyblue::]%s[-:-:-] ", d.modelId))
	d.SetInputCapture(d.keyboard)

	cfg, ok := ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		d.AddItem(bedrockErrPanel("Could not get AWS session"), 0, 1, true)
		return nil
	}
	d.cfg = cfg

	infoP := bedrockPanel(" [aqua::b]▸ MODEL INFO[-:-:-] ", bedrockLoadingText)
	capP := bedrockPanel(" [aqua::b]▸ CAPABILITIES[-:-:-] ", bedrockLoadingText)
	metricsP := bedrockPanel(" [aqua::b]▸ METRICS[-:-:-]  [darkgray::]24h  ⟳ 60s[-:-:-] ", bedrockLoadingText)
	d.metricsPanel = metricsP

	d.focusable = []tview.Primitive{infoP, capP, metricsP}

	for _, p := range d.focusable {
		d.attachNavCapture(p.(*tview.TextView))
	}

	leftCol := tview.NewFlex().SetDirection(tview.FlexRow)
	leftCol.AddItem(infoP, 0, 1, false)
	leftCol.AddItem(metricsP, 0, 1, false)

	rightCol := tview.NewFlex().SetDirection(tview.FlexRow)
	rightCol.AddItem(capP, 0, 1, false)

	prettyBody := tview.NewFlex().SetDirection(tview.FlexColumn)
	prettyBody.AddItem(leftCol, 0, 1, false)
	prettyBody.AddItem(rightCol, 0, 1, false)
	d.prettyBody = prettyBody

	jsonTV := tview.NewTextView()
	jsonTV.SetDynamicColors(true)
	jsonTV.SetBorder(true)
	jsonTV.SetBorderColor(tcell.ColorSteelBlue)
	jsonTV.SetTitle(" [aqua::b]▸ JSON[-:-:-] ")
	jsonTV.SetTitleAlign(tview.AlignLeft)
	jsonTV.SetTextColor(tcell.ColorWhite)
	jsonTV.SetScrollable(true)
	jsonTV.SetWrap(false)
	jsonTV.SetBackgroundColor(tcell.ColorDefault)
	jsonTV.SetText(bedrockLoadingText)
	jsonTV.SetInputCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		switch evt.Key() {
		case tcell.KeyEscape:
			d.app.PrevCmd(evt)
			return nil
		case tcell.KeyRune:
			if evt.Rune() == 'j' {
				d.toggleJSON()
				return nil
			}
		}
		return evt
	})
	d.jsonBody = jsonTV

	d.AddItem(prettyBody, 0, 1, true)

	go func() {
		detail := aws.GetBedrockModel(cfg, d.modelId)
		d.app.QueueUpdateDraw(func() {
			if detail == nil {
				infoP.SetText("  [red::]Could not load model " + d.modelId + "[-:-:-]\n")
				return
			}
			d.detail = detail
			d.SetTitle(fmt.Sprintf(
				" [aqua::b]Bedrock[-:-:-]  [yellow::b]%s[-:-:-]  [lightskyblue::]%s[-:-:-] ",
				detail.ProviderName, detail.ModelName,
			))
			infoP.SetText(bedrockInfoContent(detail))
			capP.SetText(bedrockCapContent(detail))
		})
	}()

	go func() {
		jsonStr := aws.GetBedrockModelJSON(cfg, d.modelId)
		d.app.QueueUpdateDraw(func() {
			jsonTV.SetText(jsonStr)
		})
	}()

	go func() {
		m := aws.GetBedrockModelMetrics(cfg, d.modelId)
		d.app.QueueUpdateDraw(func() {
			metricsP.SetText(bedrockMetricsContent(m))
		})
	}()

	return nil
}

func (d *BedrockDetail) Start() {
	if len(d.focusable) > 0 {
		d.focusIdx = 0
		d.app.Application.SetFocus(d.focusable[0])
	}
	if d.metricsPanel != nil {
		ctx, cancel := context.WithCancel(context.Background())
		d.cancelMon = cancel
		go d.pollMetrics(ctx)
	}
}

func (d *BedrockDetail) Stop() {
	if d.cancelMon != nil {
		d.cancelMon()
		d.cancelMon = nil
	}
}

func (d *BedrockDetail) pollMetrics(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m := aws.GetBedrockModelMetrics(d.cfg, d.modelId)
			d.app.QueueUpdateDraw(func() {
				d.metricsPanel.SetText(bedrockMetricsContent(m))
			})
		}
	}
}

func (d *BedrockDetail) nextFocus() {
	if len(d.focusable) == 0 {
		return
	}
	d.focusIdx = (d.focusIdx + 1) % len(d.focusable)
	d.app.Application.SetFocus(d.focusable[d.focusIdx])
}

func (d *BedrockDetail) prevFocus() {
	if len(d.focusable) == 0 {
		return
	}
	d.focusIdx = (d.focusIdx - 1 + len(d.focusable)) % len(d.focusable)
	d.app.Application.SetFocus(d.focusable[d.focusIdx])
}

func (d *BedrockDetail) attachNavCapture(tv *tview.TextView) {
	tv.SetInputCapture(func(evt *tcell.EventKey) *tcell.EventKey {
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
		case tcell.KeyRune:
			if evt.Rune() == 'j' {
				d.toggleJSON()
				return nil
			}
		}
		return evt
	})
}

func bedrockInfoContent(m *aws.BedrockModelDetailResp) string {
	var b strings.Builder
	bedrockKV(&b, "Model ID", "[yellow::]"+m.ModelId+"[-:-:-]")
	bedrockKV(&b, "Name", bedrockDash(m.ModelName))
	bedrockKV(&b, "Provider", bedrockDash(m.ProviderName))
	streaming := "[darkgray::]No[-:-:-]"
	if m.StreamingSupported {
		streaming = "[green::]Yes[-:-:-]"
	}
	bedrockKV(&b, "Streaming", streaming)
	return b.String()
}

func bedrockMetricsContent(m *aws.BedrockMetricsResp) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  [darkgray::]last 24 hours[-:-:-]\n\n")

	if m.InvocationsOK {
		bedrockKV(&b, "Invocations", fmt.Sprintf("[white::]%.0f[-:-:-]", m.Invocations))
	} else {
		bedrockKV(&b, "Invocations", "[darkgray::]no data[-:-:-]")
	}

	if m.LatencyOK {
		bedrockKV(&b, "Avg Latency", fmt.Sprintf("[white::]%.0f ms[-:-:-]", m.LatencyAvgMs))
	} else {
		bedrockKV(&b, "Avg Latency", "[darkgray::]no data[-:-:-]")
	}

	fmt.Fprintf(&b, "\n")

	if m.TokensOK {
		bedrockKV(&b, "Input Tokens", fmt.Sprintf("[white::]%.0f[-:-:-]", m.InputTokens))
		bedrockKV(&b, "Output Tokens", fmt.Sprintf("[white::]%.0f[-:-:-]", m.OutputTokens))
	} else {
		bedrockKV(&b, "Input Tokens", "[darkgray::]no data[-:-:-]")
		bedrockKV(&b, "Output Tokens", "[darkgray::]no data[-:-:-]")
	}

	fmt.Fprintf(&b, "\n")

	if m.ErrorsOK {
		clrClient := "green"
		if m.ClientErrors > 0 {
			clrClient = "red"
		}
		clrServer := "green"
		if m.ServerErrors > 0 {
			clrServer = "red"
		}
		bedrockKV(&b, "Client Errors", fmt.Sprintf("[%s::]%.0f[-:-:-]", clrClient, m.ClientErrors))
		bedrockKV(&b, "Server Errors", fmt.Sprintf("[%s::]%.0f[-:-:-]", clrServer, m.ServerErrors))
	} else {
		bedrockKV(&b, "Client Errors", "[darkgray::]no data[-:-:-]")
		bedrockKV(&b, "Server Errors", "[darkgray::]no data[-:-:-]")
	}

	if !m.InvocationsOK && !m.LatencyOK && !m.TokensOK && !m.ErrorsOK {
		fmt.Fprintf(&b, "\n  [darkgray::]Enable model invocation logging[-:-:-]\n")
		fmt.Fprintf(&b, "  [darkgray::]in Bedrock settings to see metrics.[-:-:-]\n")
	}

	return b.String()
}

func bedrockCapContent(m *aws.BedrockModelDetailResp) string {
	var b strings.Builder

	fmt.Fprintf(&b, "  [lightskyblue::]Input Modalities[-:-:-]\n")
	for _, mod := range m.InputModalities {
		fmt.Fprintf(&b, "    [white::]%s[-:-:-]\n", mod)
	}
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "  [lightskyblue::]Output Modalities[-:-:-]\n")
	for _, mod := range m.OutputModalities {
		fmt.Fprintf(&b, "    [white::]%s[-:-:-]\n", mod)
	}
	fmt.Fprintf(&b, "\n")

	if len(m.InferenceTypes) > 0 {
		fmt.Fprintf(&b, "  [lightskyblue::]Inference Types[-:-:-]\n")
		for _, t := range m.InferenceTypes {
			fmt.Fprintf(&b, "    [white::]%s[-:-:-]\n", t)
		}
		fmt.Fprintf(&b, "\n")
	}

	if len(m.Customizations) > 0 {
		fmt.Fprintf(&b, "  [lightskyblue::]Customizations[-:-:-]\n")
		for _, c := range m.Customizations {
			fmt.Fprintf(&b, "    [white::]%s[-:-:-]\n", c)
		}
	}

	return b.String()
}

func bedrockPanel(title, content string) *tview.TextView {
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

func bedrockErrPanel(msg string) *tview.TextView {
	return bedrockPanel(" [red::]Error[-:-:-] ", "  [red::]"+msg+"[-:-:-]\n")
}

func bedrockKV(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "  [lightskyblue::]%-18s[-:-:-]  %s\n", key, value)
}

func bedrockDash(s string) string {
	if s == "" {
		return "[darkgray::]—[-:-:-]"
	}
	return s
}

func (d *BedrockDetail) Name() string { return "BedrockDetail" }

func (d *BedrockDetail) Hints() model.MenuHints {
	return model.MenuHints{
		{Mnemonic: "j", Description: "JSON", Visible: true},
		{Mnemonic: "tab", Description: "Next Panel", Visible: true},
		{Mnemonic: "esc", Description: "Back", Visible: true},
		{Mnemonic: "⟳ 60s", Description: "Metrics refresh", Visible: true},
	}
}

func (d *BedrockDetail) toggleJSON() {
	d.jsonMode = !d.jsonMode
	d.RemoveItemAtIndex(0)
	name := d.modelId
	if d.detail != nil && d.detail.ModelName != "" {
		name = d.detail.ModelName
	}
	if d.jsonMode {
		d.SetTitle(fmt.Sprintf(
			" [aqua::b]Bedrock[-:-:-]  [yellow::b]%s[-:-:-]  [darkgray::]JSON  j:pretty[-:-:-] ",
			name,
		))
		d.AddItem(d.jsonBody, 0, 1, true)
		d.app.Application.SetFocus(d.jsonBody)
	} else {
		d.SetTitle(fmt.Sprintf(
			" [aqua::b]Bedrock[-:-:-]  [yellow::b]%s[-:-:-]  [lightskyblue::]%s[-:-:-] ",
			name, d.modelId,
		))
		d.AddItem(d.prettyBody, 0, 1, true)
		if len(d.focusable) > 0 {
			d.app.Application.SetFocus(d.focusable[d.focusIdx])
		}
	}
}

func (d *BedrockDetail) keyboard(evt *tcell.EventKey) *tcell.EventKey {
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
	case tcell.KeyRune:
		if evt.Rune() == 'j' {
			d.toggleJSON()
			return nil
		}
	}
	return evt
}
