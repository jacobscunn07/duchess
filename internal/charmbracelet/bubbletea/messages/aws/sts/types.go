package sts

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type IGetCallerIdentityAPI interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
	GetRegion() string
}

type GetCallerIdentityAPI struct {
	sts *sts.Client
	cfg *aws.Config
}

func NewGetCallerIdentityAPI(cfg aws.Config) *GetCallerIdentityAPI {
	return &GetCallerIdentityAPI{
		sts: sts.NewFromConfig(cfg),
		cfg: &cfg,
	}
}

func (api GetCallerIdentityAPI) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	return api.sts.GetCallerIdentity(ctx, params, optFns...)
}

func (api GetCallerIdentityAPI) GetRegion() string {
	return api.cfg.Region
}
