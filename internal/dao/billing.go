package dao

import (
	"context"
	"fmt"

	awsV2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/one2nc/cloudlens/internal"
	"github.com/one2nc/cloudlens/internal/aws"
	"github.com/rs/zerolog/log"
)

type Billing struct {
	Accessor
	ctx context.Context
}

func (b *Billing) Init(ctx context.Context) {
	b.ctx = ctx
}

func (b *Billing) List(ctx context.Context) ([]Object, error) {
	cfg, ok := ctx.Value(internal.KeySession).(awsV2.Config)
	if !ok {
		log.Err(fmt.Errorf("conversion err: Expected awsV2.Config but got %v", cfg))
	}
	services, err := aws.GetBillingByService(cfg)
	if err != nil {
		return nil, err
	}
	objs := make([]Object, len(services))
	for i, s := range services {
		objs[i] = s
	}
	return objs, nil
}

func (b *Billing) Get(ctx context.Context, path string) (Object, error) {
	return nil, nil
}

func (b *Billing) Describe(serviceName string) (string, error) {
	return serviceName, nil
}
