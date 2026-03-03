package session_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	smithy "github.com/aws/smithy-go"

	session "github.com/jacobscunn07/duchess/internal/aws"
)

// mockAPIError implements smithy.APIError for testing
type mockAPIError struct {
	code    string
	message string
}

func (m *mockAPIError) ErrorCode() string        { return m.code }
func (m *mockAPIError) ErrorMessage() string     { return m.message }
func (m *mockAPIError) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }
func (m *mockAPIError) Error() string            { return m.code + ": " + m.message }

// TestClassifyCredentialError_InvalidTokenError verifies that ssocreds.InvalidTokenError
// returns a message containing "aws sso login --profile X"
func TestClassifyCredentialError_InvalidTokenError(t *testing.T) {
	err := &ssocreds.InvalidTokenError{Err: errors.New("token expired")}
	result := session.ClassifyCredentialError(err, "myprofile")
	if result == nil {
		t.Fatal("expected error, got nil")
	}
	msg := result.Error()
	if !strings.Contains(msg, "aws sso login --profile myprofile") {
		t.Errorf("expected message to contain 'aws sso login --profile myprofile', got: %q", msg)
	}
}

// TestClassifyCredentialError_ExpiredTokenException verifies that ExpiredTokenException
// returns a message containing "credentials expired"
func TestClassifyCredentialError_ExpiredTokenException(t *testing.T) {
	apiErr := &mockAPIError{code: "ExpiredTokenException", message: "token is expired"}
	result := session.ClassifyCredentialError(apiErr, "myprofile")
	if result == nil {
		t.Fatal("expected error, got nil")
	}
	msg := result.Error()
	if !strings.Contains(msg, "credentials expired") {
		t.Errorf("expected message to contain 'credentials expired', got: %q", msg)
	}
}

// TestClassifyCredentialError_InvalidClientTokenId verifies that InvalidClientTokenId
// returns a message containing "invalid credentials" and "access key"
func TestClassifyCredentialError_InvalidClientTokenId(t *testing.T) {
	apiErr := &mockAPIError{code: "InvalidClientTokenId", message: "invalid token id"}
	result := session.ClassifyCredentialError(apiErr, "myprofile")
	if result == nil {
		t.Fatal("expected error, got nil")
	}
	msg := result.Error()
	if !strings.Contains(msg, "invalid credentials") {
		t.Errorf("expected message to contain 'invalid credentials', got: %q", msg)
	}
	if !strings.Contains(msg, "access key") {
		t.Errorf("expected message to contain 'access key', got: %q", msg)
	}
}

// TestClassifyCredentialError_AuthFailure verifies that AuthFailure
// returns a message containing "invalid credentials" and "access key"
func TestClassifyCredentialError_AuthFailure(t *testing.T) {
	apiErr := &mockAPIError{code: "AuthFailure", message: "auth failure"}
	result := session.ClassifyCredentialError(apiErr, "myprofile")
	if result == nil {
		t.Fatal("expected error, got nil")
	}
	msg := result.Error()
	if !strings.Contains(msg, "invalid credentials") {
		t.Errorf("expected message to contain 'invalid credentials', got: %q", msg)
	}
	if !strings.Contains(msg, "access key") {
		t.Errorf("expected message to contain 'access key', got: %q", msg)
	}
}

// TestClassifyCredentialError_AccessDenied verifies that AccessDenied
// returns a message containing "access denied"
func TestClassifyCredentialError_AccessDenied(t *testing.T) {
	apiErr := &mockAPIError{code: "AccessDenied", message: "access denied"}
	result := session.ClassifyCredentialError(apiErr, "myprofile")
	if result == nil {
		t.Fatal("expected error, got nil")
	}
	msg := result.Error()
	if !strings.Contains(msg, "access denied") {
		t.Errorf("expected message to contain 'access denied', got: %q", msg)
	}
}

// TestClassifyCredentialError_GenericError verifies that a generic error
// returns a wrapped error (not a stack trace)
func TestClassifyCredentialError_GenericError(t *testing.T) {
	generic := errors.New("some unexpected error")
	result := session.ClassifyCredentialError(generic, "myprofile")
	if result == nil {
		t.Fatal("expected error, got nil")
	}
	msg := result.Error()
	if !strings.Contains(msg, "authentication failed") {
		t.Errorf("expected message to contain 'authentication failed', got: %q", msg)
	}
	// Ensure it's a wrapped error (unwrappable), not a raw stack trace
	if !errors.Is(result, generic) {
		t.Errorf("expected result to wrap the original error via errors.Is")
	}
}

// TestClassifyCredentialError_Nil verifies that nil returns nil
func TestClassifyCredentialError_Nil(t *testing.T) {
	result := session.ClassifyCredentialError(nil, "myprofile")
	if result != nil {
		t.Errorf("expected nil for nil input, got: %v", result)
	}
}

// TestNewAWSConfig_EmptyProfile verifies no panic and region check works with empty profile
func TestNewAWSConfig_EmptyProfile(t *testing.T) {
	// With profile="" and region="us-east-1", should either succeed (uses default chain)
	// or return an error — but should NOT panic.
	ctx := context.Background()
	_, err := session.NewAWSConfig(ctx, "", "us-east-1")
	// We don't assert success or failure here because the default chain
	// may or may not have credentials in the CI environment.
	// The test verifies the call does not panic.
	_ = err
}

// TestNewAWSConfig_NonexistentProfile verifies error for unknown profile
func TestNewAWSConfig_NonexistentProfile(t *testing.T) {
	ctx := context.Background()
	_, err := session.NewAWSConfig(ctx, "nonexistent-test-profile-xyz", "us-east-1")
	if err == nil {
		t.Fatal("expected error for nonexistent profile, got nil")
	}
}
