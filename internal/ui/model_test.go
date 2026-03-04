package ui

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/jacobscunn07/duchess/internal/config"
)

// helpers to build a test rootModel without real AWS credentials
func newTestModel() rootModel {
	cfg := &config.Config{
		Profile:         "test-profile",
		Region:          "us-east-1",
		RefreshInterval: 30,
	}
	awsCfg := aws.Config{}
	return NewRootModel(context.Background(), cfg, awsCfg)
}

// TestNewRootModelDefaultDimensions verifies constructor sets safe default dimensions.
func TestNewRootModelDefaultDimensions(t *testing.T) {
	m := newTestModel()
	assert.Equal(t, 80, m.width, "default width should be 80")
	assert.Equal(t, 24, m.height, "default height should be 24")
}

// TestUpdateWindowSizeMsg verifies WindowSizeMsg sets width and height.
func TestUpdateWindowSizeMsg(t *testing.T) {
	m := newTestModel()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	rm := result.(rootModel)
	assert.Equal(t, 120, rm.width)
	assert.Equal(t, 40, rm.height)
}

// TestUpdateQuitKey verifies "q" key returns a non-nil cmd (tea.Quit).
func TestUpdateQuitKey(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	assert.NotNil(t, cmd, "q key should return a Quit cmd")
	// Execute the cmd and check it produces a QuitMsg
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg, "q should return QuitMsg")
}

// TestUpdateCtrlCKey verifies ctrl+c returns a non-nil cmd (tea.Quit).
func TestUpdateCtrlCKey(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.NotNil(t, cmd, "ctrl+c should return a Quit cmd")
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg, "ctrl+c should return QuitMsg")
}

// TestUpdateIdentityLoadedMsg verifies state becomes stateReady and fields are set.
func TestUpdateIdentityLoadedMsg(t *testing.T) {
	m := newTestModel()
	msg := identityLoadedMsg{account: "123456789012", arn: "arn:aws:iam::123456789012:user/test"}
	result, _ := m.Update(msg)
	rm := result.(rootModel)
	assert.Equal(t, stateReady, rm.state)
	assert.Equal(t, "123456789012", rm.account)
	assert.Equal(t, "arn:aws:iam::123456789012:user/test", rm.arn)
}

// TestUpdateIdentityErrMsg verifies state becomes stateError and err is set.
func TestUpdateIdentityErrMsg(t *testing.T) {
	m := newTestModel()
	testErr := errors.New("credential error")
	msg := identityErrMsg{err: testErr}
	result, _ := m.Update(msg)
	rm := result.(rootModel)
	assert.Equal(t, stateError, rm.state)
	assert.Equal(t, testErr, rm.err)
}

// TestViewZeroWidthReturnsEmpty verifies View() returns "" when width=0.
func TestViewZeroWidthReturnsEmpty(t *testing.T) {
	m := newTestModel()
	m.width = 0
	assert.Equal(t, "", m.View(), "View with width=0 should return empty string")
}

// TestViewDefaultDimensionsNonEmpty verifies View() returns non-empty with default 80x24.
func TestViewDefaultDimensionsNonEmpty(t *testing.T) {
	m := newTestModel()
	assert.NotEqual(t, "", m.View(), "View with default 80x24 should return non-empty string")
}

// TestUpdateTickMsg verifies tickMsg updates m.now.
func TestUpdateTickMsg(t *testing.T) {
	m := newTestModel()
	now := time.Now()
	result, _ := m.Update(tickMsg(now))
	rm := result.(rootModel)
	assert.Equal(t, now, rm.now)
}
