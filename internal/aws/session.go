// Package session provides AWS session factory functions and credential error classification.
// The package name is "session" (not "aws") to avoid shadowing the SDK's "aws" package.
package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	smithy "github.com/aws/smithy-go"
)

// NewAWSConfig loads an AWS SDK configuration for the given profile and region.
// If profile is empty, the SDK's default credential chain is used.
// If region is empty, the region from the profile's config is used; if no region
// is available, an error instructing the user to pass --region is returned.
func NewAWSConfig(ctx context.Context, profile, region string) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{}

	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, ClassifyCredentialError(err, profile)
	}

	if cfg.Region == "" {
		return aws.Config{}, fmt.Errorf(
			"no region configured for profile %q — use --region flag or set region in ~/.duchess/config",
			profile,
		)
	}

	return cfg, nil
}

// GetCallerIdentity calls STS GetCallerIdentity and returns the account ID and ARN.
func GetCallerIdentity(ctx context.Context, cfg aws.Config) (account, arn string, err error) {
	client := sts.NewFromConfig(cfg)
	out, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", "", err
	}
	return aws.ToString(out.Account), aws.ToString(out.Arn), nil
}

// ClassifyCredentialError converts raw AWS SDK errors into human-readable errors
// with actionable messages. The ssocreds.InvalidTokenError check must come BEFORE
// the smithy.APIError check because InvalidTokenError does not implement APIError.
//
// Error taxonomy:
//   - ssocreds.InvalidTokenError  → "SSO session expired. Run: aws sso login --profile X"
//   - ExpiredTokenException       → "credentials expired for profile X"
//   - InvalidClientTokenId        → "invalid credentials for profile X. Check your access key"
//   - AuthFailure                 → "invalid credentials for profile X. Check your access key"
//   - AccessDenied                → "access denied for profile X. Check IAM permissions"
//   - anything else               → wrapped error with "authentication failed" prefix
func ClassifyCredentialError(err error, profile string) error {
	if err == nil {
		return nil
	}

	// Check ssocreds.InvalidTokenError FIRST — it is a struct, not a smithy.APIError.
	var invalidToken *ssocreds.InvalidTokenError
	if errors.As(err, &invalidToken) {
		return fmt.Errorf("SSO session expired. Run: aws sso login --profile %s", profile)
	}

	// Check smithy.APIError for AWS API-level error codes.
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "ExpiredTokenException":
			return fmt.Errorf("credentials expired for profile %q. Refresh your credentials and try again.", profile)
		case "InvalidClientTokenId", "AuthFailure":
			return fmt.Errorf("invalid credentials for profile %q. Check your access key and secret key.", profile)
		case "AccessDenied":
			return fmt.Errorf("access denied for profile %q. Check IAM permissions.", profile)
		}
	}

	// Fall through: wrap unknown errors with context.
	return fmt.Errorf("authentication failed for profile %q: %w", profile, err)
}
