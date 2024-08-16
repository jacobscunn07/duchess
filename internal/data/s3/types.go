package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Api struct {
	client *s3.Client
}

func NewApi(client *s3.Client) *Api {
	return &Api{
		client: client,
	}
}

func (api *Api) GetObject(bucket, key string) (*GetObjectOutput, error) {
	result, err := api.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return &GetObjectOutput{
		Bucket:   bucket,
		Key:      key,
		Metadata: result.Metadata,
		Contents: result.Body,
	}, nil
}

type GetObjectOutput struct {
	Bucket   string
	Key      string
	Metadata map[string]string
	Contents io.ReadCloser
}
