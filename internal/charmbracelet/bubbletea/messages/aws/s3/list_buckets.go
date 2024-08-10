package s3

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	tea "github.com/charmbracelet/bubbletea"
)

func ListBucketsQuery(ctx context.Context, api IListBucketsAPI) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		var msg ListBucketsQueryMessage
		result, _ := api.ListBuckets(ctx, &s3.ListBucketsInput{})

		for _, b := range result.Buckets {
			msg.Buckets = append(msg.Buckets, ListBucketsQueryMessageBucketDetails{
				CreationDate: *b.CreationDate,
				Name:         *b.Name,
			})
		}

		return msg
	})
}

type ListBucketsQueryMessage struct {
	Buckets []ListBucketsQueryMessageBucketDetails
	Error   error
}

type ListBucketsQueryMessageBucketDetails struct {
	CreationDate time.Time
	Name         string
}
