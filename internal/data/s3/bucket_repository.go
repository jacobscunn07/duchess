package s3

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewBucketRepository(cfg aws.Config) BucketRepository {
	return BucketRepository{
		s3: s3.NewFromConfig(cfg),
	}
}

type BucketRepository struct {
	s3 *s3.Client
}

func (r *BucketRepository) List() (*Bucket, error) {
	b := Bucket{}

	return &b, nil
}
