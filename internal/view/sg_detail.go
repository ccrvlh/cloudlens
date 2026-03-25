package view

import (
	"context"
	"fmt"

	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/one2nc/cloudlens/internal"
	"github.com/one2nc/cloudlens/internal/aws"
	"github.com/one2nc/cloudlens/internal/model"
	awsV2 "github.com/aws/aws-sdk-go-v2/aws"
)

type SgDetail struct {
	*tview.Flex
	groupId string
	app     *App
}

func NewSgDetail(app *App, groupId string) *SgDetail {
	return &SgDetail{
		Flex:    tview.NewFlex(),
		groupId: groupId,
		app:     app,
	}
}

func (s *SgDetail) Init(ctx context.Context) error {
	s.SetDirection(tview.FlexColumn)
	s.SetBorder(true)
	s.SetBorderColor(tcell.ColorLightSkyBlue)
	s.SetTitle(fmt.Sprintf(" Security Group: %s ", s.groupId))
	s.SetInputCapture(s.keyboard)

	cfg, ok := ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		errView := tview.NewTextView().SetText("Error: Could not get AWS config")
		s.AddItem(errView, 0, 1, true)
		return nil
	}

	sgData := aws.GetSingleSecurityGroup(cfg, s.groupId)
	if sgData == nil {
		errView := tview.NewTextView().SetText(fmt.Sprintf("Error: Failed to load security group data for %s", s.groupId))
		s.AddItem(errView, 0, 1, true)
		return nil
	}

	infoText := fmt.Sprintf(
		" [aqua::b]Group ID[-:-:-]\n %s\n\n [aqua::b]Name[-:-:-]\n %s\n\n [aqua::b]VPC[-:-:-]\n %s\n\n [aqua::b]Owner[-:-:-]\n %s\n\n [aqua::b]Description[-:-:-]\n %s",
		sgData.GroupId, sgData.GroupName, sgData.VpcId, sgData.OwnerId, sgData.Description,
	)
	infoView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(infoText)
	infoView.SetBorder(true)
	infoView.SetBorderColor(tcell.ColorDarkSlateGray)
	infoView.SetTitle(" Details ")
	infoView.SetBorderPadding(0, 0, 1, 1)

	table := tview.NewTable()
	table.SetBorder(true)
	table.SetBorderColor(tcell.ColorDarkSlateGray)
	table.SetTitle(fmt.Sprintf(" Rules (%d ingress, %d egress) ", len(sgData.IpPermissions), len(sgData.IpPermissionsEgress)))
	table.SetFixed(1, 0)
	table.SetSelectable(true, false)
	table.SetBorderPadding(0, 0, 1, 1)

	headers := []string{"TYPE", "PROTOCOL", "PORTS", "SOURCE / DESTINATION", "DESCRIPTION"}
	for col, h := range headers {
		table.SetCell(0, col, tview.NewTableCell(h).
			SetTextColor(tcell.ColorBeige).
			SetAttributes(tcell.AttrBold).
			SetExpansion(1).
			SetAlign(tview.AlignLeft))
	}

	row := 1
	for _, perm := range sgData.IpPermissions {
		row = addSgRuleRows(table, row, "Ingress", perm, tcell.ColorLightGreen)
	}
	for _, perm := range sgData.IpPermissionsEgress {
		row = addSgRuleRows(table, row, "Egress", perm, tcell.ColorLightSalmon)
	}

	s.AddItem(infoView, 36, 0, false)
	s.AddItem(table, 0, 1, true)

	return nil
}

// addSgRuleRows adds one row per source/destination within a permission rule.
func addSgRuleRows(table *tview.Table, startRow int, ruleType string, perm aws.IpPermission, color tcell.Color) int {
	protocol := derefString(perm.IpProtocol)
	if protocol == "-1" {
		protocol = "All"
	}

	portRange := "All"
	if perm.FromPort != nil && perm.ToPort != nil {
		if *perm.FromPort == *perm.ToPort {
			portRange = fmt.Sprintf("%d", *perm.FromPort)
		} else {
			portRange = fmt.Sprintf("%d-%d", *perm.FromPort, *perm.ToPort)
		}
	} else if perm.FromPort != nil {
		portRange = fmt.Sprintf("%d+", *perm.FromPort)
	}

	type sourceEntry struct{ cidr, desc string }
	var sources []sourceEntry

	for _, r := range perm.IpRanges {
		sources = append(sources, sourceEntry{r.CidrIp, r.Description})
	}
	for _, g := range perm.UserIdGroupPairs {
		sources = append(sources, sourceEntry{g.GroupId, g.Description})
	}
	for _, p := range perm.PrefixListIds {
		sources = append(sources, sourceEntry{p.PrefixListId, ""})
	}
	if len(sources) == 0 {
		sources = []sourceEntry{{"—", ""}}
	}

	row := startRow
	for _, src := range sources {
		values := []string{ruleType, protocol, portRange, src.cidr, src.desc}
		for col, val := range values {
			table.SetCell(row, col, tview.NewTableCell(val).
				SetTextColor(color).
				SetExpansion(1).
				SetAlign(tview.AlignLeft))
		}
		row++
	}
	return row
}

func (s *SgDetail) Name() string {
	return "SgDetail"
}

func (s *SgDetail) Start() {}

func (s *SgDetail) Stop() {}

func (s *SgDetail) Hints() model.MenuHints {
	return model.MenuHints{}
}

func (s *SgDetail) keyboard(evt *tcell.EventKey) *tcell.EventKey {
	if evt.Key() == tcell.KeyEscape {
		s.app.PrevCmd(evt)
		return nil
	}
	return evt
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
