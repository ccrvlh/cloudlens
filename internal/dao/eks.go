package dao

import (
	"context"
	"fmt"

	awsV2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/one2nc/cloudlens/internal"
	"github.com/one2nc/cloudlens/internal/aws"
	"github.com/rs/zerolog/log"
)

type EKSClusters struct {
	Accessor
	ctx context.Context
}

func (e *EKSClusters) Init(ctx context.Context) {
	e.ctx = ctx
}

func (e *EKSClusters) List(ctx context.Context) ([]Object, error) {
	cfg, ok := ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		log.Err(fmt.Errorf("conversion err: Expected awsV2.Config but got %v", cfg))
	}
	clusters, err := aws.GetEKSClusters(cfg)
	if err != nil {
		return nil, err
	}
	objs := make([]Object, len(clusters))
	for i, c := range clusters {
		objs[i] = c
	}
	return objs, nil
}

func (e *EKSClusters) Get(ctx context.Context, path string) (Object, error) {
	return nil, nil
}

func (e *EKSClusters) Describe(clusterName string) (string, error) {
	cfg, ok := e.ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		errMsg := fmt.Sprintf("conversion err: Expected awsV2.Config but got %v", cfg)
		log.Err(fmt.Errorf(errMsg))
		return "", fmt.Errorf(errMsg)
	}
	res := aws.GetEKSClusterJSON(cfg, clusterName)
	return fmt.Sprintf("%v", res), nil
}
