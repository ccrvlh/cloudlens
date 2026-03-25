package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/rs/zerolog/log"
)

// GetBillingByService fetches blended cost for the current and previous month,
// grouped by AWS service. Cost Explorer is a global API that always uses us-east-1.
func GetBillingByService(cfg aws.Config) ([]BillingServiceResp, error) {
	ceClient := costexplorer.NewFromConfig(cfg, func(o *costexplorer.Options) {
		o.Region = "us-east-1"
	})

	now := time.Now().UTC()
	startCurrent := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	startPrev := startCurrent.AddDate(0, -1, 0)

	currentResult, err := ceClient.GetCostAndUsage(context.Background(), &costexplorer.GetCostAndUsageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(startCurrent.Format("2006-01-02")),
			End:   aws.String(now.Format("2006-01-02")),
		},
		Granularity: cetypes.GranularityMonthly,
		GroupBy: []cetypes.GroupDefinition{
			{Type: cetypes.GroupDefinitionTypeDimension, Key: aws.String("SERVICE")},
		},
		Metrics: []string{"BlendedCost"},
	})
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Error getting current month billing: %v", err))
		return nil, err
	}

	prevResult, err := ceClient.GetCostAndUsage(context.Background(), &costexplorer.GetCostAndUsageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(startPrev.Format("2006-01-02")),
			End:   aws.String(startCurrent.Format("2006-01-02")),
		},
		Granularity: cetypes.GranularityMonthly,
		GroupBy: []cetypes.GroupDefinition{
			{Type: cetypes.GroupDefinitionTypeDimension, Key: aws.String("SERVICE")},
		},
		Metrics: []string{"BlendedCost"},
	})
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Error getting previous month billing: %v", err))
		return nil, err
	}

	type costEntry struct{ Amount, Unit string }
	currentCosts := make(map[string]costEntry)
	if len(currentResult.ResultsByTime) > 0 {
		for _, group := range currentResult.ResultsByTime[0].Groups {
			if len(group.Keys) > 0 {
				m := group.Metrics["BlendedCost"]
				currentCosts[group.Keys[0]] = costEntry{
					Amount: aws.ToString(m.Amount),
					Unit:   aws.ToString(m.Unit),
				}
			}
		}
	}

	prevCosts := make(map[string]costEntry)
	if len(prevResult.ResultsByTime) > 0 {
		for _, group := range prevResult.ResultsByTime[0].Groups {
			if len(group.Keys) > 0 {
				m := group.Metrics["BlendedCost"]
				prevCosts[group.Keys[0]] = costEntry{
					Amount: aws.ToString(m.Amount),
					Unit:   aws.ToString(m.Unit),
				}
			}
		}
	}

	seen := make(map[string]bool)
	for k := range currentCosts {
		seen[k] = true
	}
	for k := range prevCosts {
		seen[k] = true
	}

	var results []BillingServiceResp
	for serviceName := range seen {
		curr := currentCosts[serviceName]
		prev := prevCosts[serviceName]

		unit := curr.Unit
		if unit == "" {
			unit = prev.Unit
		}
		currentAmount := curr.Amount
		if currentAmount == "" {
			currentAmount = "0.0000"
		}
		prevAmount := prev.Amount
		if prevAmount == "" {
			prevAmount = "0.0000"
		}

		results = append(results, BillingServiceResp{
			ServiceName:       serviceName,
			CurrentMonthCost:  currentAmount,
			PreviousMonthCost: prevAmount,
			Unit:              unit,
		})
	}

	return results, nil
}

// GetServiceMonthlyCosts fetches monthly costs for a specific service over the
// last `months` complete months, oldest first.
func GetServiceMonthlyCosts(cfg aws.Config, serviceName string, months int) ([]BillingMonthlyResp, error) {
	ceClient := costexplorer.NewFromConfig(cfg, func(o *costexplorer.Options) {
		o.Region = "us-east-1"
	})

	now := time.Now().UTC()
	end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	start := end.AddDate(0, -months, 0)

	result, err := ceClient.GetCostAndUsage(context.Background(), &costexplorer.GetCostAndUsageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(start.Format("2006-01-02")),
			End:   aws.String(end.Format("2006-01-02")),
		},
		Granularity: cetypes.GranularityMonthly,
		Filter: &cetypes.Expression{
			Dimensions: &cetypes.DimensionValues{
				Key:    cetypes.DimensionService,
				Values: []string{serviceName},
			},
		},
		Metrics: []string{"BlendedCost"},
	})
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Error getting monthly costs for %s: %v", serviceName, err))
		return nil, err
	}

	var monthly []BillingMonthlyResp
	for _, period := range result.ResultsByTime {
		month := ""
		if period.TimePeriod != nil {
			t, _ := time.Parse("2006-01-02", aws.ToString(period.TimePeriod.Start))
			month = t.Format("Jan 2006")
		}
		amount := "0.0000"
		unit := "USD"
		if m, ok := period.Total["BlendedCost"]; ok {
			amount = aws.ToString(m.Amount)
			unit = aws.ToString(m.Unit)
		}
		monthly = append(monthly, BillingMonthlyResp{
			Month:  month,
			Amount: amount,
			Unit:   unit,
		})
	}
	return monthly, nil
}
