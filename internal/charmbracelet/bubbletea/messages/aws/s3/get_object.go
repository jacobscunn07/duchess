package s3

import (
	"context"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jacobscunn07/duchess/internal/data/s3"
)

func GetObjectQuery(ctx context.Context, api s3.Api, bucket, key string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		result, _ := api.GetObject(bucket, key)

		return GetObjectQueryMessage{
			Bucket:   result.Bucket,
			Key:      result.Key,
			Contents: result.Contents,
		}
	})
}

type GetObjectQueryMessage struct {
	Bucket   string
	Key      string
	Contents io.ReadCloser
}
