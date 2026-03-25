package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/rs/zerolog/log"
)

func GetBedrockModels(cfg aws.Config) ([]BedrockModelResp, error) {
	client := bedrock.NewFromConfig(cfg)
	result, err := client.ListFoundationModels(context.Background(), &bedrock.ListFoundationModelsInput{})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error listing Bedrock foundation models: %v", err))
		return nil, err
	}

	var models []BedrockModelResp
	for _, m := range result.ModelSummaries {
		inputMods := make([]string, len(m.InputModalities))
		for i, mod := range m.InputModalities {
			inputMods[i] = string(mod)
		}
		outputMods := make([]string, len(m.OutputModalities))
		for i, mod := range m.OutputModalities {
			outputMods[i] = string(mod)
		}
		streaming := "No"
		if m.ResponseStreamingSupported != nil && *m.ResponseStreamingSupported {
			streaming = "Yes"
		}
		models = append(models, BedrockModelResp{
			ModelId:          aws.ToString(m.ModelId),
			ModelName:        aws.ToString(m.ModelName),
			ProviderName:     aws.ToString(m.ProviderName),
			InputModalities:  strings.Join(inputMods, ", "),
			OutputModalities: strings.Join(outputMods, ", "),
			Streaming:        streaming,
		})
	}
	return models, nil
}

func GetBedrockModel(cfg aws.Config, modelId string) *BedrockModelDetailResp {
	client := bedrock.NewFromConfig(cfg)
	result, err := client.GetFoundationModel(context.Background(), &bedrock.GetFoundationModelInput{
		ModelIdentifier: &modelId,
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error describing Bedrock model %s: %v", modelId, err))
		return nil
	}
	m := result.ModelDetails

	inputMods := make([]string, len(m.InputModalities))
	for i, mod := range m.InputModalities {
		inputMods[i] = string(mod)
	}
	outputMods := make([]string, len(m.OutputModalities))
	for i, mod := range m.OutputModalities {
		outputMods[i] = string(mod)
	}
	inferenceTypes := make([]string, len(m.InferenceTypesSupported))
	for i, t := range m.InferenceTypesSupported {
		inferenceTypes[i] = string(t)
	}
	customizations := make([]string, len(m.CustomizationsSupported))
	for i, c := range m.CustomizationsSupported {
		customizations[i] = string(c)
	}
	streaming := false
	if m.ResponseStreamingSupported != nil {
		streaming = *m.ResponseStreamingSupported
	}

	return &BedrockModelDetailResp{
		ModelId:            aws.ToString(m.ModelId),
		ModelName:          aws.ToString(m.ModelName),
		ProviderName:       aws.ToString(m.ProviderName),
		InputModalities:    inputMods,
		OutputModalities:   outputMods,
		StreamingSupported: streaming,
		InferenceTypes:     inferenceTypes,
		Customizations:     customizations,
	}
}

func GetBedrockModelMetrics(cfg aws.Config, modelId string) *BedrockMetricsResp {
	result := &BedrockMetricsResp{}
	cwClient := cloudwatch.NewFromConfig(cfg)
	now := time.Now()
	start := now.Add(-24 * time.Hour)
	dim := []cwtypes.Dimension{{Name: aws.String("ModelId"), Value: aws.String(modelId)}}

	getSum := func(metric string) (float64, bool) {
		out, err := cwClient.GetMetricStatistics(context.Background(), &cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("AWS/Bedrock"),
			MetricName: aws.String(metric),
			Dimensions: dim,
			StartTime:  &start,
			EndTime:    &now,
			Period:     aws.Int32(86400),
			Statistics: []cwtypes.Statistic{cwtypes.StatisticSum},
		})
		if err != nil || len(out.Datapoints) == 0 {
			return 0, false
		}
		return aws.ToFloat64(out.Datapoints[0].Sum), true
	}

	getAvg := func(metric string) (float64, bool) {
		out, err := cwClient.GetMetricStatistics(context.Background(), &cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String("AWS/Bedrock"),
			MetricName: aws.String(metric),
			Dimensions: dim,
			StartTime:  &start,
			EndTime:    &now,
			Period:     aws.Int32(86400),
			Statistics: []cwtypes.Statistic{cwtypes.StatisticAverage},
		})
		if err != nil || len(out.Datapoints) == 0 {
			return 0, false
		}
		return aws.ToFloat64(out.Datapoints[0].Average), true
	}

	result.Invocations, result.InvocationsOK = getSum("Invocations")
	result.LatencyAvgMs, result.LatencyOK = getAvg("InvocationLatency")
	result.ClientErrors, result.ErrorsOK = getSum("InvocationClientErrors")
	result.ServerErrors, _ = getSum("InvocationServerErrors")
	result.ErrorsOK = result.ErrorsOK || result.ServerErrors > 0
	result.InputTokens, result.TokensOK = getSum("InputTokenCount")
	result.OutputTokens, _ = getSum("OutputTokenCount")

	return result
}

func GetBedrockModelJSON(cfg aws.Config, modelId string) string {
	client := bedrock.NewFromConfig(cfg)
	result, err := client.GetFoundationModel(context.Background(), &bedrock.GetFoundationModelInput{
		ModelIdentifier: &modelId,
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error describing Bedrock model %s: %v", modelId, err))
		return ""
	}
	r, _ := json.MarshalIndent(result, "", " ")
	return string(r)
}
