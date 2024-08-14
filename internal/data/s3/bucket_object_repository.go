package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type BucketObjectRepository struct {
	bucket string
	s3     *s3.Client
}

func NewBucketObjectRepository(cfg aws.Config, bucket string) BucketObjectRepository {
	return BucketObjectRepository{
		bucket: bucket,
		s3:     s3.NewFromConfig(cfg),
	}
}

func (m *BucketObjectRepository) List() ([]BucketObject, error) {
	result, err := m.s3.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(m.bucket),
	})
	if err != nil {
		return nil, err
	}

	bucketObjects := []BucketObject{}

	for _, o := range result.Contents {

		bucketObjects = append(bucketObjects, BucketObject{
			Key: *o.Key,
		})
	}

	return bucketObjects, nil
}
