package dao

import (
	"context"
	"fmt"

	awsV2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/one2nc/cloudlens/internal"
	"github.com/one2nc/cloudlens/internal/aws"
	"github.com/rs/zerolog/log"
)

type BedrockModels struct {
	Accessor
	ctx context.Context
}

func (b *BedrockModels) Init(ctx context.Context) {
	b.ctx = ctx
}

func (b *BedrockModels) List(ctx context.Context) ([]Object, error) {
	cfg, ok := ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		log.Err(fmt.Errorf("conversion err: Expected awsV2.Config but got %v", cfg))
	}
	models, err := aws.GetBedrockModels(cfg)
	if err != nil {
		return nil, err
	}
	objs := make([]Object, len(models))
	for i, m := range models {
		objs[i] = m
	}
	return objs, nil
}

func (b *BedrockModels) Get(ctx context.Context, path string) (Object, error) {
	return nil, nil
}

func (b *BedrockModels) Describe(modelId string) (string, error) {
	cfg, ok := b.ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		errMsg := fmt.Sprintf("conversion err: Expected awsV2.Config but got %v", cfg)
		log.Err(fmt.Errorf(errMsg))
		return "", fmt.Errorf(errMsg)
	}
	res := aws.GetBedrockModelJSON(cfg, modelId)
	return fmt.Sprintf("%v", res), nil
}
