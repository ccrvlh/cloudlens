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

const eksLoadingText = "  [darkgray::]loading…[-:-:-]\n"

type EKSDetail struct {
	*tview.Flex
	clusterName string
	app         *App
	cfg         awsV2.Config
	cluster     *aws.EKSClusterDetailResp

	focusable []tview.Primitive
	focusIdx  int

	jsonMode   bool
	prettyBody tview.Primitive
	jsonBody   tview.Primitive
}

func NewEKSDetail(app *App, clusterName string) *EKSDetail {
	return &EKSDetail{
		Flex:        tview.NewFlex(),
		clusterName: clusterName,
		app:         app,
	}
}

func (e *EKSDetail) Init(ctx context.Context) error {
	e.SetDirection(tview.FlexRow)
	e.SetBorder(true)
	e.SetBorderColor(tcell.ColorLightSkyBlue)
	e.SetTitle(fmt.Sprintf(" [aqua::b]EKS[-:-:-]  [lightskyblue::]%s[-:-:-] ", e.clusterName))
	e.SetInputCapture(e.keyboard)

	cfg, ok := ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		e.AddItem(eksErrPanel("Could not get AWS session"), 0, 1, true)
		return nil
	}
	e.cfg = cfg

	identityP := eksPanel(" [aqua::b]▸ IDENTITY[-:-:-] ", eksLoadingText)
	networkP := eksNetworkingPanelShell(e)
	ngTable := eksNodeGroupTableShell(e)
	addonsP := eksPanel(" [aqua::b]▸ ADD-ONS[-:-:-] ", eksLoadingText)
	tagsP := eksPanel(" [aqua::b]▸ TAGS[-:-:-] ", eksLoadingText)

	e.focusable = []tview.Primitive{identityP, networkP, ngTable, addonsP, tagsP}

	for _, p := range []tview.Primitive{identityP, addonsP, tagsP} {
		e.attachNavCapture(p.(*tview.TextView))
	}

	leftCol := tview.NewFlex().SetDirection(tview.FlexRow)
	leftCol.AddItem(identityP, 0, 2, false)
	leftCol.AddItem(addonsP, 0, 1, false)

	rightCol := tview.NewFlex().SetDirection(tview.FlexRow)
	rightCol.AddItem(networkP, 0, 2, false)
	rightCol.AddItem(ngTable, 0, 2, false)
	rightCol.AddItem(tagsP, 0, 1, false)

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
	jsonTV.SetText(eksLoadingText)
	jsonTV.SetInputCapture(e.panelCapture(nil))
	e.jsonBody = jsonTV

	e.AddItem(prettyBody, 0, 1, true)

	go func() {
		cluster := aws.GetEKSCluster(cfg, e.clusterName)
		e.app.QueueUpdateDraw(func() {
			if cluster == nil {
				identityP.SetText("  [red::]Could not load cluster " + e.clusterName + "[-:-:-]\n")
				return
			}
			e.cluster = cluster
			e.SetTitle(fmt.Sprintf(
				" [aqua::b]EKS[-:-:-]  [yellow::b]%s[-:-:-]  [lightskyblue::]k8s %s[-:-:-] ",
				cluster.Name, cluster.Version,
			))
			identityP.SetText(eksIdentityContent(cluster))
			networkP.SetText(eksNetworkingContent(cluster))
			tagsP.SetTitle(fmt.Sprintf(" [aqua::b]▸ TAGS[-:-:-]  [darkgray::](%d)[-:-:-] ", len(cluster.Tags)))
			tagsP.SetText(eksTagsContent(cluster))
		})
	}()

	go func() {
		nodeGroups := aws.GetEKSNodeGroups(cfg, e.clusterName)
		e.app.QueueUpdateDraw(func() {
			eksPopulateNodeGroupTable(ngTable, nodeGroups)
		})
	}()

	go func() {
		addons := aws.GetEKSAddons(cfg, e.clusterName)
		e.app.QueueUpdateDraw(func() {
			addonsP.SetTitle(fmt.Sprintf(" [aqua::b]▸ ADD-ONS[-:-:-]  [darkgray::](%d)[-:-:-] ", len(addons)))
			addonsP.SetText(eksAddonsContent(addons))
		})
	}()

	go func() {
		jsonStr := aws.GetEKSClusterJSON(cfg, e.clusterName)
		e.app.QueueUpdateDraw(func() {
			jsonTV.SetText(jsonStr)
		})
	}()

	return nil
}

func (e *EKSDetail) Start() {
	if len(e.focusable) > 0 {
		e.focusIdx = 0
		e.app.Application.SetFocus(e.focusable[0])
	}
}

func (e *EKSDetail) Stop() {}

func (e *EKSDetail) nextFocus() {
	if len(e.focusable) == 0 {
		return
	}
	e.focusIdx = (e.focusIdx + 1) % len(e.focusable)
	e.app.Application.SetFocus(e.focusable[e.focusIdx])
}

func (e *EKSDetail) prevFocus() {
	if len(e.focusable) == 0 {
		return
	}
	e.focusIdx = (e.focusIdx - 1 + len(e.focusable)) % len(e.focusable)
	e.app.Application.SetFocus(e.focusable[e.focusIdx])
}

func (e *EKSDetail) panelCapture(extra func(*tcell.EventKey) *tcell.EventKey) func(*tcell.EventKey) *tcell.EventKey {
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

func (e *EKSDetail) attachNavCapture(tv *tview.TextView) {
	tv.SetInputCapture(e.panelCapture(nil))
}

func eksNetworkingPanelShell(d *EKSDetail) *tview.TextView {
	tv := eksPanel(" [aqua::b]▸ NETWORKING[-:-:-] ", eksLoadingText)
	tv.SetInputCapture(d.panelCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		if evt.Key() == tcell.KeyRune && evt.Rune() == 'v' {
			d.app.inject(NewVPC("vpc"))
			return nil
		}
		return evt
	}))
	return tv
}

func eksNodeGroupTableShell(d *EKSDetail) *tview.Table {
	table := tview.NewTable()
	table.SetBorder(true)
	table.SetBorderColor(tcell.ColorSteelBlue)
	table.SetBorderFocusColor(tcell.ColorDeepSkyBlue)
	table.SetTitle(" [aqua::b]▸ NODE GROUPS[-:-:-]  [darkgray::]loading…[-:-:-] ")
	table.SetTitleAlign(tview.AlignLeft)
	table.SetSelectable(true, false)
	table.SetBackgroundColor(tcell.ColorDefault)
	table.SetCell(0, 0, tview.NewTableCell("  loading…").SetTextColor(tcell.ColorDarkGray))
	table.SetInputCapture(d.panelCapture(nil))
	return table
}

func eksPopulateNodeGroupTable(table *tview.Table, nodeGroups []aws.EKSNodeGroupResp) {
	table.Clear()
	count := len(nodeGroups)
	table.SetTitle(fmt.Sprintf(" [aqua::b]▸ NODE GROUPS[-:-:-]  [darkgray::](%d)[-:-:-] ", count))
	if count == 0 {
		table.SetCell(0, 0, tview.NewTableCell("  none").SetTextColor(tcell.ColorDarkGray))
		return
	}
	for i, ng := range nodeGroups {
		dot, _ := eksStatusDot(ng.Status)
		instanceTypes := strings.Join(ng.InstanceTypes, ", ")
		if instanceTypes == "" {
			instanceTypes = "—"
		}
		scaling := fmt.Sprintf("%d / %d–%d", ng.DesiredSize, ng.MinSize, ng.MaxSize)
		table.SetCell(i, 0,
			tview.NewTableCell("  "+ng.Name).SetTextColor(tcell.ColorYellow).SetExpansion(2))
		table.SetCell(i, 1,
			tview.NewTableCell(dot+ng.Status+"  ").SetTextColor(tcell.ColorWhite).SetExpansion(1))
		table.SetCell(i, 2,
			tview.NewTableCell(instanceTypes+"  ").SetTextColor(tcell.ColorLightSkyBlue).SetExpansion(2))
		table.SetCell(i, 3,
			tview.NewTableCell(ng.CapacityType+"  ").SetTextColor(tcell.ColorDarkGray).SetExpansion(1))
		table.SetCell(i, 4,
			tview.NewTableCell(scaling+"  ").SetTextColor(tcell.ColorWhite).SetExpansion(1))
	}
}

func eksIdentityContent(c *aws.EKSClusterDetailResp) string {
	dot, clr := eksStatusDot(c.Status)
	stateText := fmt.Sprintf("%s[%s::b]%s[-:-:-]", dot, clr, c.Status)
	var b strings.Builder
	eksKV(&b, "Name", "[yellow::]"+c.Name+"[-:-:-]")
	eksKV(&b, "Status", stateText)
	eksKV(&b, "K8s Version", c.Version)
	eksKV(&b, "ARN", eksTrunc(c.Arn, 52))
	eksKV(&b, "Role ARN", eksTrunc(c.RoleArn, 52))
	eksKV(&b, "Created At", eksDash(c.CreatedAt))
	return b.String()
}

func eksNetworkingContent(c *aws.EKSClusterDetailResp) string {
	var b strings.Builder
	eksKV(&b, "VPC ID", "[yellow::]"+eksDash(c.VpcId)+"[-:-:-]")
	eksKV(&b, "Endpoint", eksTrunc(eksDash(c.Endpoint), 52))
	eksKV(&b, "Public Access", eksBool(c.PublicAccess))
	eksKV(&b, "Private Access", eksBool(c.PrivateAccess))
	eksKV(&b, "Cluster SG", "[yellow::]"+eksDash(c.ClusterSGId)+"[-:-:-]")
	fmt.Fprintf(&b, "\n  [lightskyblue::]Subnets (%d)[-:-:-]\n", len(c.SubnetIds))
	for _, s := range c.SubnetIds {
		fmt.Fprintf(&b, "    [white::]%s[-:-:-]\n", s)
	}
	fmt.Fprintf(&b, "\n  [lightskyblue::]Security Groups (%d)[-:-:-]\n", len(c.SecurityGroupIds))
	for _, sg := range c.SecurityGroupIds {
		fmt.Fprintf(&b, "    [white::]%s[-:-:-]\n", sg)
	}
	fmt.Fprintf(&b, "\n  [darkgray::]v: VPC list[-:-:-]\n")
	return b.String()
}

func eksAddonsContent(addons []aws.EKSAddonResp) string {
	var b strings.Builder
	if len(addons) == 0 {
		fmt.Fprintf(&b, "  [darkgray::]none[-:-:-]\n")
		return b.String()
	}
	for _, a := range addons {
		dot, _ := eksStatusDot(eksAddonStatusNorm(a.Status))
		fmt.Fprintf(&b, "  %s[white::]%-28s[-:-:-]  [lightskyblue::]%s[-:-:-]  [darkgray::]%s[-:-:-]\n",
			dot, a.Name, a.Version, a.Status)
	}
	return b.String()
}

func eksTagsContent(c *aws.EKSClusterDetailResp) string {
	var b strings.Builder
	if len(c.Tags) == 0 {
		fmt.Fprintf(&b, "  [darkgray::]none[-:-:-]\n")
		return b.String()
	}
	keys := make([]string, 0, len(c.Tags))
	for k := range c.Tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "  [lightskyblue::]%-22s[-:-:-]  [white::]%s[-:-:-]\n", k, c.Tags[k])
	}
	return b.String()
}

func eksPanel(title, content string) *tview.TextView {
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

func eksErrPanel(msg string) *tview.TextView {
	return eksPanel(" [red::]Error[-:-:-] ", "  [red::]"+msg+"[-:-:-]\n")
}

func eksKV(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "  [lightskyblue::]%-18s[-:-:-]  %s\n", key, value)
}

func eksStatusDot(status string) (dot, colorName string) {
	switch status {
	case "ACTIVE":
		return "[green::]◉[-:-:-] ", "green"
	case "CREATING", "UPDATING":
		return "[yellow::]◎[-:-:-] ", "yellow"
	case "DELETING":
		return "[red::]◯[-:-:-] ", "red"
	case "FAILED", "CREATE_FAILED", "DELETE_FAILED", "DEGRADED", "UPDATE_FAILED":
		return "[red::]✕[-:-:-] ", "red"
	default:
		return "[gray::]◌[-:-:-] ", "gray"
	}
}

func eksAddonStatusNorm(s string) string {
	switch s {
	case "ACTIVE":
		return "ACTIVE"
	case "CREATING", "UPDATING":
		return "CREATING"
	case "CREATE_FAILED", "DELETE_FAILED", "DEGRADED", "UPDATE_FAILED":
		return "FAILED"
	case "DELETING":
		return "DELETING"
	default:
		return s
	}
}

func eksBool(v bool) string {
	if v {
		return "[green::]✓ yes[-:-:-]"
	}
	return "[darkgray::]✗ no[-:-:-]"
}

func eksDash(s string) string {
	if s == "" {
		return "[darkgray::]—[-:-:-]"
	}
	return s
}

func eksTrunc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "[darkgray::]…[-:-:-]"
}

func (e *EKSDetail) Name() string { return "EKSDetail" }

func (e *EKSDetail) Hints() model.MenuHints {
	return model.MenuHints{
		{Mnemonic: "j", Description: "JSON", Visible: true},
		{Mnemonic: "v", Description: "VPCs", Visible: true},
		{Mnemonic: "tab", Description: "Next Panel", Visible: true},
		{Mnemonic: "esc", Description: "Back", Visible: true},
	}
}

func (e *EKSDetail) toggleJSON() {
	e.jsonMode = !e.jsonMode
	e.RemoveItemAtIndex(0)
	if e.jsonMode {
		e.SetTitle(fmt.Sprintf(
			" [aqua::b]EKS[-:-:-]  [yellow::b]%s[-:-:-]  [darkgray::]JSON  j:pretty[-:-:-] ",
			e.clusterName,
		))
		e.AddItem(e.jsonBody, 0, 1, true)
		e.app.Application.SetFocus(e.jsonBody)
	} else {
		e.SetTitle(fmt.Sprintf(
			" [aqua::b]EKS[-:-:-]  [yellow::b]%s[-:-:-]  [lightskyblue::]k8s %s[-:-:-] ",
			e.clusterName,
			func() string {
				if e.cluster != nil {
					return e.cluster.Version
				}
				return ""
			}(),
		))
		e.AddItem(e.prettyBody, 0, 1, true)
		if len(e.focusable) > 0 {
			e.app.Application.SetFocus(e.focusable[e.focusIdx])
		}
	}
}

func (e *EKSDetail) keyboard(evt *tcell.EventKey) *tcell.EventKey {
	return e.panelCapture(nil)(evt)
}
