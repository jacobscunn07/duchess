package s3

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jacobscunn07/duchess/internal/data/s3"
)

func ListBucketObjectsQuery(ctx context.Context, repository s3.BucketObjectRepository) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		var msg ListBucketObjectsQueryMessage
		result, _ := repository.List()

		msg.Objects = []string{}
		for _, o := range result {
			msg.Objects = append(msg.Objects, o.Key)
		}

		return msg
	})
}

type ListBucketObjectsQueryMessage struct {
	Objects []string
}
