package view

import (
	"context"
	"fmt"
	"sort"
	"strings"

	awsV2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/one2nc/cloudlens/internal"
	"github.com/one2nc/cloudlens/internal/aws"
	"github.com/one2nc/cloudlens/internal/model"
)

const apigwLoadingText = "  [darkgray::]loading…[-:-:-]\n"

type APIGWDetail struct {
	*tview.Flex
	apiId  string
	app    *App
	cfg    awsV2.Config
	detail *aws.APIGatewayDetailResp

	focusable []tview.Primitive
	focusIdx  int

	jsonMode   bool
	prettyBody tview.Primitive
	jsonBody   tview.Primitive
}

func NewAPIGWDetail(app *App, apiId string) *APIGWDetail {
	return &APIGWDetail{
		Flex:  tview.NewFlex(),
		apiId: apiId,
		app:   app,
	}
}

func (a *APIGWDetail) Init(ctx context.Context) error {
	a.SetDirection(tview.FlexRow)
	a.SetBorder(true)
	a.SetBorderColor(tcell.ColorLightSkyBlue)
	a.SetTitle(fmt.Sprintf(" [aqua::b]API Gateway[-:-:-]  [lightskyblue::]%s[-:-:-] ", a.apiId))
	a.SetInputCapture(a.keyboard)

	cfg, ok := ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		a.AddItem(apigwErrPanel("Could not get AWS session"), 0, 1, true)
		return nil
	}
	a.cfg = cfg

	identityP := apigwPanel(" [aqua::b]▸ IDENTITY[-:-:-] ", apigwLoadingText)
	stagesP := apigwPanel(" [aqua::b]▸ STAGES[-:-:-] ", apigwLoadingText)
	tagsP := apigwPanel(" [aqua::b]▸ TAGS[-:-:-] ", apigwLoadingText)

	a.focusable = []tview.Primitive{identityP, stagesP, tagsP}

	for _, p := range a.focusable {
		a.attachNavCapture(p.(*tview.TextView))
	}

	leftCol := tview.NewFlex().SetDirection(tview.FlexRow)
	leftCol.AddItem(identityP, 0, 1, false)

	rightCol := tview.NewFlex().SetDirection(tview.FlexRow)
	rightCol.AddItem(stagesP, 0, 2, false)
	rightCol.AddItem(tagsP, 0, 1, false)

	prettyBody := tview.NewFlex().SetDirection(tview.FlexColumn)
	prettyBody.AddItem(leftCol, 0, 1, false)
	prettyBody.AddItem(rightCol, 0, 1, false)
	a.prettyBody = prettyBody

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
	jsonTV.SetText(apigwLoadingText)
	jsonTV.SetInputCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		switch evt.Key() {
		case tcell.KeyEscape:
			a.app.PrevCmd(evt)
			return nil
		case tcell.KeyRune:
			if evt.Rune() == 'j' {
				a.toggleJSON()
				return nil
			}
		}
		return evt
	})
	a.jsonBody = jsonTV

	a.AddItem(prettyBody, 0, 1, true)

	go func() {
		detail := aws.GetAPIGateway(cfg, a.apiId)
		a.app.QueueUpdateDraw(func() {
			if detail == nil {
				identityP.SetText("  [red::]Could not load API Gateway " + a.apiId + "[-:-:-]\n")
				return
			}
			a.detail = detail
			a.SetTitle(fmt.Sprintf(
				" [aqua::b]API Gateway[-:-:-]  [yellow::b]%s[-:-:-]  [lightskyblue::]%s[-:-:-] ",
				detail.Name, detail.ID,
			))
			identityP.SetText(apigwIdentityContent(detail))
			stagesP.SetTitle(fmt.Sprintf(" [aqua::b]▸ STAGES[-:-:-]  [darkgray::](%d)[-:-:-] ", len(detail.Stages)))
			stagesP.SetText(apigwStagesContent(detail))
			tagsP.SetTitle(fmt.Sprintf(" [aqua::b]▸ TAGS[-:-:-]  [darkgray::](%d)[-:-:-] ", len(detail.Tags)))
			tagsP.SetText(apigwTagsContent(detail))
		})
	}()

	go func() {
		jsonStr := aws.GetAPIGatewayJSON(cfg, a.apiId)
		a.app.QueueUpdateDraw(func() {
			jsonTV.SetText(jsonStr)
		})
	}()

	return nil
}

func (a *APIGWDetail) Start() {
	if len(a.focusable) > 0 {
		a.focusIdx = 0
		a.app.Application.SetFocus(a.focusable[0])
	}
}

func (a *APIGWDetail) Stop() {}

func (a *APIGWDetail) nextFocus() {
	if len(a.focusable) == 0 {
		return
	}
	a.focusIdx = (a.focusIdx + 1) % len(a.focusable)
	a.app.Application.SetFocus(a.focusable[a.focusIdx])
}

func (a *APIGWDetail) prevFocus() {
	if len(a.focusable) == 0 {
		return
	}
	a.focusIdx = (a.focusIdx - 1 + len(a.focusable)) % len(a.focusable)
	a.app.Application.SetFocus(a.focusable[a.focusIdx])
}

func (a *APIGWDetail) attachNavCapture(tv *tview.TextView) {
	tv.SetInputCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		switch evt.Key() {
		case tcell.KeyEscape:
			a.app.PrevCmd(evt)
			return nil
		case tcell.KeyTab:
			a.nextFocus()
			return nil
		case tcell.KeyBacktab:
			a.prevFocus()
			return nil
		case tcell.KeyRune:
			if evt.Rune() == 'j' {
				a.toggleJSON()
				return nil
			}
		}
		return evt
	})
}

func apigwIdentityContent(d *aws.APIGatewayDetailResp) string {
	var b strings.Builder
	apigwKV(&b, "API ID", "[yellow::]"+d.ID+"[-:-:-]")
	apigwKV(&b, "Name", apigwDash(d.Name))
	apigwKV(&b, "Endpoint Type", apigwDash(d.EndpointType))
	apigwKV(&b, "API Key Source", apigwDash(d.APIKeySource))
	apigwKV(&b, "Created Date", apigwDash(d.CreatedDate))
	apigwKV(&b, "Description", apigwDash(d.Description))
	return b.String()
}

func apigwStagesContent(d *aws.APIGatewayDetailResp) string {
	var b strings.Builder
	if len(d.Stages) == 0 {
		fmt.Fprintf(&b, "  [darkgray::]none[-:-:-]\n")
		return b.String()
	}
	for _, s := range d.Stages {
		fmt.Fprintf(&b, "  [yellow::b]%s[-:-:-]\n", s.StageName)
		if s.Description != "" {
			fmt.Fprintf(&b, "    [darkgray::]%s[-:-:-]\n", s.Description)
		}
		apigwKVIndent(&b, "Deployment ID", apigwDash(s.DeploymentID))
		apigwKVIndent(&b, "Created", apigwDash(s.CreatedDate))
		apigwKVIndent(&b, "Last Updated", apigwDash(s.LastUpdated))
		fmt.Fprintf(&b, "\n")
	}
	return b.String()
}

func apigwTagsContent(d *aws.APIGatewayDetailResp) string {
	var b strings.Builder
	if len(d.Tags) == 0 {
		fmt.Fprintf(&b, "  [darkgray::]none[-:-:-]\n")
		return b.String()
	}
	keys := make([]string, 0, len(d.Tags))
	for k := range d.Tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "  [lightskyblue::]%-22s[-:-:-]  [white::]%s[-:-:-]\n", k, d.Tags[k])
	}
	return b.String()
}

func apigwPanel(title, content string) *tview.TextView {
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

func apigwErrPanel(msg string) *tview.TextView {
	return apigwPanel(" [red::]Error[-:-:-] ", "  [red::]"+msg+"[-:-:-]\n")
}

func apigwKV(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "  [lightskyblue::]%-18s[-:-:-]  %s\n", key, value)
}

func apigwKVIndent(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "    [lightskyblue::]%-16s[-:-:-]  %s\n", key, value)
}

func apigwDash(s string) string {
	if s == "" {
		return "[darkgray::]—[-:-:-]"
	}
	return s
}

func (a *APIGWDetail) Name() string { return "APIGWDetail" }

func (a *APIGWDetail) Hints() model.MenuHints {
	return model.MenuHints{
		{Mnemonic: "j", Description: "JSON", Visible: true},
		{Mnemonic: "tab", Description: "Next Panel", Visible: true},
		{Mnemonic: "esc", Description: "Back", Visible: true},
	}
}

func (a *APIGWDetail) toggleJSON() {
	a.jsonMode = !a.jsonMode
	a.RemoveItemAtIndex(0)
	name := a.apiId
	if a.detail != nil && a.detail.Name != "" {
		name = a.detail.Name
	}
	if a.jsonMode {
		a.SetTitle(fmt.Sprintf(
			" [aqua::b]API Gateway[-:-:-]  [yellow::b]%s[-:-:-]  [darkgray::]JSON  j:pretty[-:-:-] ",
			name,
		))
		a.AddItem(a.jsonBody, 0, 1, true)
		a.app.Application.SetFocus(a.jsonBody)
	} else {
		a.SetTitle(fmt.Sprintf(
			" [aqua::b]API Gateway[-:-:-]  [yellow::b]%s[-:-:-]  [lightskyblue::]%s[-:-:-] ",
			name, a.apiId,
		))
		a.AddItem(a.prettyBody, 0, 1, true)
		if len(a.focusable) > 0 {
			a.app.Application.SetFocus(a.focusable[a.focusIdx])
		}
	}
}

func (a *APIGWDetail) keyboard(evt *tcell.EventKey) *tcell.EventKey {
	switch evt.Key() {
	case tcell.KeyEscape:
		a.app.PrevCmd(evt)
		return nil
	case tcell.KeyTab:
		a.nextFocus()
		return nil
	case tcell.KeyBacktab:
		a.prevFocus()
		return nil
	case tcell.KeyRune:
		if evt.Rune() == 'j' {
			a.toggleJSON()
			return nil
		}
	}
	return evt
}
