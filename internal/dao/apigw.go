package dao

import (
	"context"
	"fmt"

	awsV2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/one2nc/cloudlens/internal"
	"github.com/one2nc/cloudlens/internal/aws"
	"github.com/rs/zerolog/log"
)

type APIGateways struct {
	Accessor
	ctx context.Context
}

func (a *APIGateways) Init(ctx context.Context) {
	a.ctx = ctx
}

func (a *APIGateways) List(ctx context.Context) ([]Object, error) {
	cfg, ok := ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		log.Err(fmt.Errorf("conversion err: Expected awsV2.Config but got %v", cfg))
	}
	apis, err := aws.GetAPIGateways(cfg)
	if err != nil {
		return nil, err
	}
	objs := make([]Object, len(apis))
	for i, a := range apis {
		objs[i] = a
	}
	return objs, nil
}

func (a *APIGateways) Get(ctx context.Context, path string) (Object, error) {
	return nil, nil
}

func (a *APIGateways) Describe(apiId string) (string, error) {
	cfg, ok := a.ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		errMsg := fmt.Sprintf("conversion err: Expected awsV2.Config but got %v", cfg)
		log.Err(fmt.Errorf(errMsg))
		return "", fmt.Errorf(errMsg)
	}
	res := aws.GetAPIGatewayJSON(cfg, apiId)
	return fmt.Sprintf("%v", res), nil
}
