package sts

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	tea "github.com/charmbracelet/bubbletea"
)

func GetCallerIdentity(ctx context.Context, api IGetCallerIdentityAPI) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		result, err := api.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})

		if err != nil {
			return GetCallerIdentityErrorMessage{Error: err}
		}

		var profile string
		if p := os.Getenv("AWS_PROFILE"); p != "" {
			profile = p
		} else {
			profile = "default"
		}

		return GetCallerIdentityMessage{
			AccountId: *result.Account,
			Arn:       *result.Arn,
			Region:    api.GetRegion(),
			Profile:   profile,
		}
	})
}

type GetCallerIdentityMessage struct {
	AccountId string
	Arn       string
	Region    string
	Profile   string
}

type GetCallerIdentityErrorMessage struct {
	Error error
}
