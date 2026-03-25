package view

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	awsV2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/one2nc/cloudlens/internal"
	"github.com/one2nc/cloudlens/internal/aws"
	"github.com/one2nc/cloudlens/internal/model"
)

const ec2LoadingText = "  [darkgray::]loading…[-:-:-]\n"

// EC2Detail shows full instance information with navigable panels.
type EC2Detail struct {
	*tview.Flex
	instanceId string
	app        *App
	cfg        awsV2.Config
	inst       *aws.EC2DetailResp

	focusable []tview.Primitive
	focusIdx  int

	monPanel  *tview.TextView
	cancelMon context.CancelFunc

	jsonMode   bool
	prettyBody tview.Primitive // the two-column panel layout
	jsonBody   tview.Primitive // raw JSON view
}

func NewEC2Detail(app *App, instanceId string) *EC2Detail {
	return &EC2Detail{
		Flex:       tview.NewFlex(),
		instanceId: instanceId,
		app:        app,
	}
}

func (e *EC2Detail) Init(ctx context.Context) error {
	e.SetDirection(tview.FlexRow)
	e.SetBorder(true)
	e.SetBorderColor(tcell.ColorLightSkyBlue)
	e.SetTitle(fmt.Sprintf(" [aqua::b]EC2[-:-:-]  [lightskyblue::]%s[-:-:-] ", e.instanceId))
	e.SetInputCapture(e.keyboard)

	cfg, ok := ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		e.AddItem(ec2ErrPanel("Could not get AWS session"), 0, 1, true)
		return nil
	}
	e.cfg = cfg

	identityP := ec2Panel(" [aqua::b]▸ IDENTITY[-:-:-] ", ec2LoadingText)
	placementP := ec2Panel(" [aqua::b]▸ PLACEMENT[-:-:-] ", ec2LoadingText)
	networkP := ec2NetworkingPanelShell(e)
	monP := ec2Panel(" [aqua::b]▸ MONITORING[-:-:-]  [darkgray::]⟳ 60s[-:-:-] ", ec2LoadingText)
	sgTable := ec2SGTableShell(e)
	tagsP := ec2Panel(" [aqua::b]▸ TAGS[-:-:-] ", ec2LoadingText)

	e.monPanel = monP

	e.focusable = []tview.Primitive{
		identityP, placementP, networkP, monP, sgTable, tagsP,
	}

	for _, p := range []tview.Primitive{identityP, placementP, monP, tagsP} {
		e.attachNavCapture(p.(*tview.TextView))
	}

	leftCol := tview.NewFlex().SetDirection(tview.FlexRow)
	leftCol.AddItem(identityP, 0, 3, false)
	leftCol.AddItem(placementP, 0, 1, false)

	rightCol := tview.NewFlex().SetDirection(tview.FlexRow)
	rightCol.AddItem(networkP, 0, 3, false)
	rightCol.AddItem(monP, 0, 3, false)
	rightCol.AddItem(sgTable, 0, 1, false)
	rightCol.AddItem(tagsP, 0, 2, false)

	prettyBody := tview.NewFlex().SetDirection(tview.FlexColumn)
	prettyBody.AddItem(leftCol, 0, 2, false)
	prettyBody.AddItem(rightCol, 0, 3, false)
	e.prettyBody = prettyBody

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
	jsonTV.SetText(ec2LoadingText)
	jsonTV.SetInputCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		switch evt.Key() {
		case tcell.KeyEscape:
			e.app.PrevCmd(evt)
			return nil
		case tcell.KeyRune:
			if evt.Rune() == 'j' {
				e.toggleJSON()
				return nil
			}
		}
		return evt
	})
	e.jsonBody = jsonTV

	e.AddItem(prettyBody, 0, 1, true)

	go func() {
		inst := aws.GetSingleInstanceDetail(cfg, e.instanceId)
		e.app.QueueUpdateDraw(func() {
			if inst == nil {
				identityP.SetText("  [red::]Could not load instance " + e.instanceId + "[-:-:-]\n")
				return
			}
			e.inst = inst
			name := inst.Name
			if name == "" {
				name = inst.InstanceId
			}
			e.SetTitle(fmt.Sprintf(
				" [aqua::b]EC2[-:-:-]  [yellow::b]%s[-:-:-]  [lightskyblue::]%s[-:-:-] ",
				name, inst.InstanceId,
			))
			identityP.SetText(ec2IdentityContent(inst))
			placementP.SetText(ec2PlacementContent(inst))
			networkP.SetText(ec2NetworkingContent(inst))
			ec2PopulateSGTable(sgTable, inst)
			tagsP.SetTitle(fmt.Sprintf(" [aqua::b]▸ TAGS[-:-:-]  [darkgray::](%d)[-:-:-] ", len(inst.Tags)))
			tagsP.SetText(ec2TagsContent(inst))
		})
	}()

	go func() {
		mon := aws.GetInstanceMonitoring(cfg, e.instanceId)
		e.app.QueueUpdateDraw(func() {
			monP.SetText(ec2MonitoringContent(mon))
		})
	}()

	go func() {
		jsonStr := aws.GetSingleInstance(cfg, e.instanceId)
		e.app.QueueUpdateDraw(func() {
			jsonTV.SetText(jsonStr)
		})
	}()

	return nil
}

// Start is called when pushed onto the page stack.
func (e *EC2Detail) Start() {
	if len(e.focusable) > 0 {
		e.focusIdx = 0
		e.app.Application.SetFocus(e.focusable[0])
	}
	if e.monPanel != nil {
		ctx, cancel := context.WithCancel(context.Background())
		e.cancelMon = cancel
		go e.pollMonitoring(ctx)
	}
}

// Stop is called when popped from the page stack.
func (e *EC2Detail) Stop() {
	if e.cancelMon != nil {
		e.cancelMon()
		e.cancelMon = nil
	}
}

func (e *EC2Detail) pollMonitoring(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mon := aws.GetInstanceMonitoring(e.cfg, e.instanceId)
			e.app.QueueUpdateDraw(func() {
				e.monPanel.SetText(ec2MonitoringContent(mon))
			})
		}
	}
}

func (e *EC2Detail) nextFocus() {
	if len(e.focusable) == 0 {
		return
	}
	e.focusIdx = (e.focusIdx + 1) % len(e.focusable)
	e.app.Application.SetFocus(e.focusable[e.focusIdx])
}

func (e *EC2Detail) prevFocus() {
	if len(e.focusable) == 0 {
		return
	}
	e.focusIdx = (e.focusIdx - 1 + len(e.focusable)) % len(e.focusable)
	e.app.Application.SetFocus(e.focusable[e.focusIdx])
}

// panelCapture returns an input capture func that handles all common keys
// (Escape, Tab, Backtab, j) and delegates anything else to extra.
func (e *EC2Detail) panelCapture(extra func(*tcell.EventKey) *tcell.EventKey) func(*tcell.EventKey) *tcell.EventKey {
	return func(evt *tcell.EventKey) *tcell.EventKey {
		switch evt.Key() {
		case tcell.KeyEscape:
			e.app.PrevCmd(evt)
			return nil
		case tcell.KeyTab:
			e.nextFocus()
			return nil
		case tcell.KeyBacktab:
			e.prevFocus()
			return nil
		case tcell.KeyRune:
			if evt.Rune() == 'j' {
				e.toggleJSON()
				return nil
			}
		}
		if extra != nil {
			return extra(evt)
		}
		return evt
	}
}

func (e *EC2Detail) attachNavCapture(tv *tview.TextView) {
	tv.SetInputCapture(e.panelCapture(nil))
}

func ec2NetworkingPanelShell(d *EC2Detail) *tview.TextView {
	tv := ec2Panel(" [aqua::b]▸ NETWORKING[-:-:-] ", ec2LoadingText)
	tv.SetInputCapture(d.panelCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		if evt.Key() == tcell.KeyRune {
			switch evt.Rune() {
			case 'v':
				d.app.inject(NewVPC("vpc"))
				return nil
			case 's':
				if d.inst != nil && d.inst.VpcId != "" {
					ctx := context.WithValue(d.app.GetContext(), internal.VpcId, d.inst.VpcId)
					d.app.SetContext(ctx)
					d.app.inject(NewSubnet("subnet"))
				}
				return nil
			}
		}
		return evt
	}))
	return tv
}

func ec2SGTableShell(d *EC2Detail) *tview.Table {
	table := tview.NewTable()
	table.SetBorder(true)
	table.SetBorderColor(tcell.ColorSteelBlue)
	table.SetBorderFocusColor(tcell.ColorDeepSkyBlue)
	table.SetTitle(" [aqua::b]▸ SECURITY GROUPS[-:-:-]  [darkgray::]loading…[-:-:-] ")
	table.SetTitleAlign(tview.AlignLeft)
	table.SetSelectable(true, false)
	table.SetBackgroundColor(tcell.ColorDefault)
	table.SetCell(0, 0, tview.NewTableCell("  loading…").SetTextColor(tcell.ColorDarkGray))

	table.SetInputCapture(d.panelCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		if evt.Key() == tcell.KeyEnter {
			if d.inst == nil {
				return nil
			}
			row, _ := table.GetSelection()
			if row < len(d.inst.SecurityGroups) {
				sgId := d.inst.SecurityGroups[row].GroupId
				d.app.inject(NewSgDetail(d.app, sgId))
			}
			return nil
		}
		return evt
	}))
	return table
}

func ec2PopulateSGTable(table *tview.Table, inst *aws.EC2DetailResp) {
	table.Clear()
	count := len(inst.SecurityGroups)
	table.SetTitle(fmt.Sprintf(" [aqua::b]▸ SECURITY GROUPS[-:-:-]  [darkgray::](%d)  ↵ view[-:-:-] ", count))
	if count == 0 {
		table.SetCell(0, 0, tview.NewTableCell("  none").SetTextColor(tcell.ColorDarkGray))
		return
	}
	for i, sg := range inst.SecurityGroups {
		table.SetCell(i, 0,
			tview.NewTableCell("  "+sg.GroupId).
				SetTextColor(tcell.ColorYellow).
				SetExpansion(1))
		table.SetCell(i, 1,
			tview.NewTableCell(sg.GroupName+"  ").
				SetTextColor(tcell.ColorWhite).
				SetExpansion(2))
	}
}

func ec2IdentityContent(inst *aws.EC2DetailResp) string {
	dot, clr := ec2StateDot(inst.InstanceState)
	stateText := fmt.Sprintf("%s[%s::b]%s[-:-:-]", dot, clr, inst.InstanceState)
	var b strings.Builder
	ec2KV(&b, "Instance ID", "[yellow::]"+inst.InstanceId+"[-:-:-]")
	ec2KV(&b, "Name", ec2Dash(inst.Name))
	ec2KV(&b, "State", stateText)
	ec2KV(&b, "Instance Type", inst.InstanceType)
	ec2KV(&b, "AMI ID", "[yellow::]"+ec2Dash(inst.ImageId)+"[-:-:-]")
	ec2KV(&b, "Key Pair", ec2Dash(inst.KeyName))
	ec2KV(&b, "Architecture", ec2Dash(inst.Architecture))
	ec2KV(&b, "Root Device", ec2RootDevice(inst))
	ec2KV(&b, "IAM Profile", ec2Trunc(ec2Dash(inst.IamProfile), 42))
	ec2KV(&b, "Launch Time", ec2Dash(inst.LaunchTime))
	return b.String()
}

func ec2PlacementContent(inst *aws.EC2DetailResp) string {
	var b strings.Builder
	ec2KV(&b, "Availability Zone", inst.AvailabilityZone)
	ec2KV(&b, "Tenancy", inst.Tenancy)
	ec2KV(&b, "Monitoring", ec2MonitoringBadge(inst.MonitoringState))
	return b.String()
}

func ec2NetworkingContent(inst *aws.EC2DetailResp) string {
	var b strings.Builder
	ec2KV(&b, "VPC ID", "[yellow::]"+ec2Dash(inst.VpcId)+"[-:-:-]")
	ec2KV(&b, "Subnet ID", "[yellow::]"+ec2Dash(inst.SubnetId)+"[-:-:-]")
	ec2KV(&b, "Private IP", ec2Dash(inst.PrivateIP))
	ec2KV(&b, "Public IP", ec2Dash(inst.PublicIP))
	ec2KV(&b, "Private DNS", ec2Trunc(ec2Dash(inst.PrivateDNS), 42))
	ec2KV(&b, "Public DNS", ec2Trunc(ec2Dash(inst.PublicDNS), 42))
	fmt.Fprintf(&b, "\n  [darkgray::]v: VPC list  s: Subnets for this VPC[-:-:-]\n")
	return b.String()
}

func ec2TagsContent(inst *aws.EC2DetailResp) string {
	var b strings.Builder
	if len(inst.Tags) == 0 {
		fmt.Fprintf(&b, "  [darkgray::]none[-:-:-]\n")
	} else {
		keys := make([]string, 0, len(inst.Tags))
		for k := range inst.Tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "  [lightskyblue::]%-22s[-:-:-]  [white::]%s[-:-:-]\n", k, inst.Tags[k])
		}
	}
	return b.String()
}

func ec2MonitoringContent(mon *aws.EC2MonitoringResp) string {
	var b strings.Builder

	ec2KV(&b, "Instance Status", ec2StatusBadge(mon.InstanceStatus))
	ec2KV(&b, "System Status", ec2StatusBadge(mon.SystemStatus))
	fmt.Fprintf(&b, "\n")

	if mon.CPUAvg1hOK {
		bar := ec2UtilBar(mon.CPUAvg1h)
		ec2KV(&b, "CPU (1h avg)", fmt.Sprintf("%s  [white::]%.1f%%[-:-:-]", bar, mon.CPUAvg1h))
	} else {
		ec2KV(&b, "CPU (1h avg)", "[darkgray::]no data[-:-:-]")
	}
	if len(mon.CPUSpark) > 0 {
		ec2KV(&b, "CPU (last 60m)", ec2Sparkline(mon.CPUSpark))
	}

	if mon.MemOK {
		bar := ec2UtilBar(mon.MemUsedPct)
		ec2KV(&b, "Mem used", fmt.Sprintf("%s  [white::]%.1f%%[-:-:-]", bar, mon.MemUsedPct))
	} else {
		ec2KV(&b, "Mem used", "[darkgray::]no data (needs CW agent)[-:-:-]")
	}
	fmt.Fprintf(&b, "\n")

	if mon.NetOK {
		ec2KV(&b, "Net In  (5m)", ec2FormatBytes(mon.NetInAvg5m))
		ec2KV(&b, "Net Out (5m)", ec2FormatBytes(mon.NetOutAvg5m))
	} else {
		ec2KV(&b, "Net In  (5m)", "[darkgray::]no data[-:-:-]")
		ec2KV(&b, "Net Out (5m)", "[darkgray::]no data[-:-:-]")
	}

	return b.String()
}

func ec2Panel(title, content string) *tview.TextView {
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

func ec2ErrPanel(msg string) *tview.TextView {
	return ec2Panel(" [red::]Error[-:-:-] ", "  [red::]"+msg+"[-:-:-]\n")
}

func ec2KV(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "  [lightskyblue::]%-18s[-:-:-]  %s\n", key, value)
}

func ec2StateDot(state string) (dot, colorName string) {
	switch state {
	case "running":
		return "[green::]◉[-:-:-] ", "green"
	case "stopped":
		return "[red::]◯[-:-:-] ", "red"
	case "pending", "stopping":
		return "[yellow::]◎[-:-:-] ", "yellow"
	case "terminated":
		return "[darkgray::]✕[-:-:-] ", "darkgray"
	default:
		return "[gray::]◌[-:-:-] ", "gray"
	}
}

func ec2StatusBadge(status string) string {
	switch status {
	case "ok":
		return "[green::]✓ ok[-:-:-]"
	case "impaired":
		return "[red::]✗ impaired[-:-:-]"
	case "initializing":
		return "[yellow::]⟳ initializing[-:-:-]"
	case "unavailable":
		return "[darkgray::]— unavailable[-:-:-]"
	default:
		return "[gray::]" + status + "[-:-:-]"
	}
}

func ec2MonitoringBadge(state string) string {
	switch state {
	case "enabled":
		return "[green::]● enabled[-:-:-]"
	case "disabled":
		return "[darkgray::]○ disabled[-:-:-]"
	default:
		return state
	}
}

// ec2UtilBar renders a 10-char bar that turns yellow at 50% and red at 80%.
func ec2UtilBar(pct float64) string {
	const width = 10
	filled := int(pct / 100.0 * width)
	if filled > width {
		filled = width
	}
	clr := "green"
	if pct >= 80 {
		clr = "red"
	} else if pct >= 50 {
		clr = "yellow"
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("[%s::]%s[-:-:-]", clr, bar)
}

// ec2Sparkline maps a slice of 0-100 values to unicode block chars.
func ec2Sparkline(data []float64) string {
	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	var sb strings.Builder
	for _, v := range data {
		idx := int(v / 100.0 * float64(len(chars)))
		if idx >= len(chars) {
			idx = len(chars) - 1
		}
		if idx < 0 {
			idx = 0
		}
		clr := "green"
		if v >= 80 {
			clr = "red"
		} else if v >= 50 {
			clr = "yellow"
		}
		sb.WriteString(fmt.Sprintf("[%s::]%c[-:-:-]", clr, chars[idx]))
	}
	return sb.String()
}

func ec2FormatBytes(b float64) string {
	switch {
	case b >= 1e9:
		return fmt.Sprintf("%.2f GB", b/1e9)
	case b >= 1e6:
		return fmt.Sprintf("%.2f MB", b/1e6)
	case b >= 1e3:
		return fmt.Sprintf("%.2f KB", b/1e3)
	default:
		return fmt.Sprintf("%.0f B", b)
	}
}

func ec2RootDevice(inst *aws.EC2DetailResp) string {
	if inst.RootDeviceName == "" {
		return "[darkgray::]—[-:-:-]"
	}
	return fmt.Sprintf("%s  [darkgray::](%s)[-:-:-]", inst.RootDeviceName, inst.RootDeviceType)
}

func ec2Dash(s string) string {
	if s == "" {
		return "[darkgray::]—[-:-:-]"
	}
	return s
}

func ec2Trunc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "[darkgray::]…[-:-:-]"
}

func (e *EC2Detail) Name() string { return "EC2Detail" }

func (e *EC2Detail) Hints() model.MenuHints {
	return model.MenuHints{
		{Mnemonic: "j", Description: "JSON", Visible: true},
		{Mnemonic: "v", Description: "VPCs", Visible: true},
		{Mnemonic: "s", Description: "Subnets", Visible: true},
		{Mnemonic: "tab", Description: "Next Panel", Visible: true},
		{Mnemonic: "esc", Description: "Back", Visible: true},
		{Mnemonic: "↵", Description: "View SG", Visible: true},
	}
}

func (e *EC2Detail) toggleJSON() {
	e.jsonMode = !e.jsonMode
	e.RemoveItemAtIndex(0)
	name := e.instanceId
	if e.inst != nil && e.inst.Name != "" {
		name = e.inst.Name
	}
	if e.jsonMode {
		e.SetTitle(fmt.Sprintf(
			" [aqua::b]EC2[-:-:-]  [yellow::b]%s[-:-:-]  [darkgray::]JSON  j:pretty[-:-:-] ",
			name,
		))
		e.AddItem(e.jsonBody, 0, 1, true)
		e.app.Application.SetFocus(e.jsonBody)
	} else {
		e.SetTitle(fmt.Sprintf(
			" [aqua::b]EC2[-:-:-]  [yellow::b]%s[-:-:-]  [lightskyblue::]%s[-:-:-] ",
			name, e.instanceId,
		))
		e.AddItem(e.prettyBody, 0, 1, true)
		if len(e.focusable) > 0 {
			e.app.Application.SetFocus(e.focusable[e.focusIdx])
		}
	}
}

func (e *EC2Detail) keyboard(evt *tcell.EventKey) *tcell.EventKey {
	switch evt.Key() {
	case tcell.KeyEscape:
		e.app.PrevCmd(evt)
		return nil
	case tcell.KeyTab:
		e.nextFocus()
		return nil
	case tcell.KeyBacktab:
		e.prevFocus()
		return nil
	case tcell.KeyRune:
		if evt.Rune() == 'j' {
			e.toggleJSON()
			return nil
		}
	}
	return evt
}
