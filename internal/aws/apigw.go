package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/one2nc/cloudlens/internal/config"
	"github.com/rs/zerolog/log"
)

func GetAPIGateways(cfg aws.Config) ([]APIGatewayResp, error) {
	client := apigateway.NewFromConfig(cfg)
	result, err := client.GetRestApis(context.Background(), &apigateway.GetRestApisInput{})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error listing API Gateways: %v", err))
		return nil, err
	}

	var apis []APIGatewayResp
	for _, item := range result.Items {
		createdDate := ""
		if item.CreatedDate != nil {
			localZone, tzErr := config.GetLocalTimeZone()
			if tzErr == nil {
				loc, _ := time.LoadLocation(localZone)
				createdDate = item.CreatedDate.In(loc).Format("Mon Jan _2 15:04:05 2006")
			} else {
				createdDate = item.CreatedDate.UTC().Format("Mon Jan _2 15:04:05 2006")
			}
		}
		endpointType := ""
		if item.EndpointConfiguration != nil && len(item.EndpointConfiguration.Types) > 0 {
			endpointType = string(item.EndpointConfiguration.Types[0])
		}
		apis = append(apis, APIGatewayResp{
			ID:           aws.ToString(item.Id),
			Name:         aws.ToString(item.Name),
			Description:  aws.ToString(item.Description),
			EndpointType: endpointType,
			CreatedDate:  createdDate,
		})
	}
	return apis, nil
}

func GetAPIGateway(cfg aws.Config, apiId string) *APIGatewayDetailResp {
	client := apigateway.NewFromConfig(cfg)

	result, err := client.GetRestApi(context.Background(), &apigateway.GetRestApiInput{
		RestApiId: &apiId,
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error getting API Gateway %s: %v", apiId, err))
		return nil
	}

	detail := &APIGatewayDetailResp{
		ID:          aws.ToString(result.Id),
		Name:        aws.ToString(result.Name),
		Description: aws.ToString(result.Description),
		APIKeySource: string(result.ApiKeySource),
		Tags:        result.Tags,
	}
	if result.EndpointConfiguration != nil && len(result.EndpointConfiguration.Types) > 0 {
		detail.EndpointType = string(result.EndpointConfiguration.Types[0])
	}
	if result.CreatedDate != nil {
		localZone, tzErr := config.GetLocalTimeZone()
		if tzErr == nil {
			loc, _ := time.LoadLocation(localZone)
			detail.CreatedDate = result.CreatedDate.In(loc).Format("Mon Jan _2 15:04:05 2006")
		} else {
			detail.CreatedDate = result.CreatedDate.UTC().Format("Mon Jan _2 15:04:05 2006")
		}
	}

	stages, err := client.GetStages(context.Background(), &apigateway.GetStagesInput{
		RestApiId: &apiId,
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error getting stages for API %s: %v", apiId, err))
	} else {
		for _, s := range stages.Item {
			stage := APIGatewayStage{
				StageName:    aws.ToString(s.StageName),
				Description:  aws.ToString(s.Description),
				DeploymentID: aws.ToString(s.DeploymentId),
			}
			if s.CreatedDate != nil {
				stage.CreatedDate = s.CreatedDate.UTC().Format("2006-01-02 15:04:05")
			}
			if s.LastUpdatedDate != nil {
				stage.LastUpdated = s.LastUpdatedDate.UTC().Format("2006-01-02 15:04:05")
			}
			detail.Stages = append(detail.Stages, stage)
		}
	}

	return detail
}

func GetAPIGatewayJSON(cfg aws.Config, apiId string) string {
	client := apigateway.NewFromConfig(cfg)
	result, err := client.GetRestApi(context.Background(), &apigateway.GetRestApiInput{
		RestApiId: &apiId,
	})
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Error getting API Gateway %s: %v", apiId, err))
		return ""
	}
	r, _ := json.MarshalIndent(result, "", " ")
	return string(r)
}
