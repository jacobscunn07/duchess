package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jacobscunn07/duchess/internal/config"
	"github.com/stretchr/testify/assert"
)

// buildTestModelForStatus creates a rootModel with the given dimensions and state
// without needing real AWS credentials or a running tea.Program.
func buildTestModelForStatus(width, height int, s state, account, arn string) rootModel {
	cfg := &config.Config{
		Profile:         "test-profile",
		Region:          "us-east-1",
		RefreshInterval: 30,
	}
	m := NewRootModel(context.Background(), cfg)
	m.width = width
	m.height = height
	m.state = s
	m.account = account
	m.arn = arn
	m.now = time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	return m
}

// TestTruncateARNShort verifies that a short ARN is returned unchanged.
func TestTruncateARNShort(t *testing.T) {
	arn := "arn:aws:iam::123456789012:user/alice"
	result := truncateARN(arn, 40)
	assert.Equal(t, arn, result, "short ARN should not be truncated")
}

// TestTruncateARNLong verifies that a long ARN is truncated to maxLen with "..." suffix.
func TestTruncateARNLong(t *testing.T) {
	arn := "arn:aws:iam::123456789012:role/very-long-role-name-that-exceeds-forty-chars"
	result := truncateARN(arn, 40)
	assert.Equal(t, 40, len(result), "truncated ARN should be exactly maxLen characters")
	assert.True(t, strings.HasSuffix(result, "..."), "truncated ARN should end with '...'")
}

// TestTruncateARNExactLength verifies ARN at exact maxLen is not truncated.
func TestTruncateARNExactLength(t *testing.T) {
	// Build an ARN exactly 40 chars long
	arn := strings.Repeat("a", 40)
	result := truncateARN(arn, 40)
	assert.Equal(t, arn, result, "ARN at exact maxLen should not be truncated")
}

// TestRenderStatusBarZeroWidthReturnsEmpty verifies the guard for zero-width renders.
func TestRenderStatusBarZeroWidth(t *testing.T) {
	m := buildTestModelForStatus(0, 24, stateLoading, "", "")
	result := renderStatusBar(m)
	assert.Equal(t, "", result, "renderStatusBar with width=0 should return empty string")
}

// TestRenderStatusBarWidth80 verifies the status bar renders to exactly width=80 display columns.
func TestRenderStatusBarWidth80(t *testing.T) {
	m := buildTestModelForStatus(80, 24, stateReady, "123456789012", "arn:aws:iam::123456789012:user/test")
	result := renderStatusBar(m)
	w := lipgloss.Width(result)
	assert.Equal(t, 80, w, "renderStatusBar width=80 should render exactly 80 display columns")
}

// TestRenderStatusBarLoadingContainsLoadingText verifies stateLoading shows "loading..." in the right group.
func TestRenderStatusBarLoadingContainsLoadingText(t *testing.T) {
	m := buildTestModelForStatus(80, 24, stateLoading, "", "")
	result := renderStatusBar(m)
	assert.Contains(t, result, "loading...", "loading state should show 'loading...' in status bar")
}

// TestRenderStatusBarLoadingContainsVersion verifies stateLoading left group has the version string.
func TestRenderStatusBarLoadingContainsVersion(t *testing.T) {
	m := buildTestModelForStatus(80, 24, stateLoading, "", "")
	result := renderStatusBar(m)
	// Strip ANSI for comparison
	plain := stripANSI(result)
	assert.Contains(t, plain, version, "status bar should contain the version string")
}

// TestRenderStatusBarReadyContainsProfile verifies stateReady left group has the profile.
func TestRenderStatusBarReadyContainsProfile(t *testing.T) {
	m := buildTestModelForStatus(80, 24, stateReady, "123456789012", "arn:aws:iam::123456789012:user/test")
	result := renderStatusBar(m)
	plain := stripANSI(result)
	assert.Contains(t, plain, "test-profile", "status bar should contain the profile name")
}

// TestRenderStatusBarReadyContainsAccount verifies stateReady right group has the account ID.
func TestRenderStatusBarReadyContainsAccount(t *testing.T) {
	m := buildTestModelForStatus(80, 24, stateReady, "123456789012", "arn:aws:iam::123456789012:user/test")
	result := renderStatusBar(m)
	plain := stripANSI(result)
	assert.Contains(t, plain, "123456789012", "status bar should contain the account ID")
}

// TestRenderStatusBarErrorState verifies stateError shows "error" in the right group.
func TestRenderStatusBarErrorState(t *testing.T) {
	m := buildTestModelForStatus(80, 24, stateError, "", "")
	result := renderStatusBar(m)
	plain := stripANSI(result)
	assert.Contains(t, plain, "error", "stateError should show 'error' in status bar")
}

// TestRenderStatusBarContainsClock verifies the clock time appears in the status bar.
func TestRenderStatusBarContainsClock(t *testing.T) {
	m := buildTestModelForStatus(80, 24, stateReady, "123456789012", "arn:aws:iam::123456789012:user/test")
	result := renderStatusBar(m)
	plain := stripANSI(result)
	// Fixed time: 14:30:45
	assert.Contains(t, plain, "14:30:45", "status bar should show the clock time in HH:MM:SS format")
}

// stripANSI removes ANSI escape sequences from a string for plain-text comparison.
// This is a simple approximation sufficient for test assertions.
func stripANSI(s string) string {
	// Use lipgloss's built-in strip function approach: render with NO_COLOR profile
	// Simpler: just check if the plain text substrings appear anywhere in the raw string
	// For our purposes we'll strip manually using a state machine.
	var out strings.Builder
	inEscape := false
	for _, ch := range s {
		if ch == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if ch == 'm' {
				inEscape = false
			}
			continue
		}
		out.WriteRune(ch)
	}
	return out.String()
}
