package sts

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	tea "github.com/charmbracelet/bubbletea"
)

func GetCallerIdentity(ctx context.Context, api IGetCallerIdentityAPI) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		result, err := api.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})

		if err != nil {
			return GetCallerIdentityErrorMessage{Error: err}
		}

		return GetCallerIdentityMessage{
			AccountId: *result.Account,
			Arn:       *result.Arn,
			Region:    api.GetRegion(),
		}
	})
}

type GetCallerIdentityMessage struct {
	AccountId string
	Arn       string
	Region    string
}

type GetCallerIdentityErrorMessage struct {
	Error error
}
