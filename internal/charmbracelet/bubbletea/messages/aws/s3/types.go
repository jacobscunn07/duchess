package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type IListBucketsAPI interface {
	ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
	GetRegion() string
}

type ListBucketsAPI struct {
	s3  *s3.Client
	cfg *aws.Config
}

func NewListBucketsAPI(cfg aws.Config) *ListBucketsAPI {
	return &ListBucketsAPI{
		s3:  s3.NewFromConfig(cfg),
		cfg: &cfg,
	}
}

func (api ListBucketsAPI) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	return api.s3.ListBuckets(ctx, params, optFns...)
}

func (api ListBucketsAPI) GetRegion() string {
	return api.cfg.Region
}
