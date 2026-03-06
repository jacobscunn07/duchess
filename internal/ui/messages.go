package ui

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// identityLoadedMsg is sent when STS GetCallerIdentity succeeds.
type identityLoadedMsg struct {
	account string
	arn     string
	cfg     aws.Config // AWS config used for the session — passed to s3Panel on init
}

// identityErrMsg is sent when STS GetCallerIdentity fails.
// err is already classified via session.ClassifyCredentialError.
type identityErrMsg struct {
	err error
}

// tickMsg is sent every second to advance the clock in the status bar.
type tickMsg time.Time

// profileSelectedMsg is sent when the user confirms a profile selection in the overlay.
type profileSelectedMsg struct {
	profile string
}
