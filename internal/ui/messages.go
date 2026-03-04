package ui

import "time"

// identityLoadedMsg is sent when STS GetCallerIdentity succeeds.
type identityLoadedMsg struct {
	account string
	arn     string
}

// identityErrMsg is sent when STS GetCallerIdentity fails.
// err is already classified via session.ClassifyCredentialError.
type identityErrMsg struct {
	err error
}

// tickMsg is sent every second to advance the clock in the status bar.
type tickMsg time.Time
