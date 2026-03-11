package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/jacobscunn07/duchess/internal/config"
)

func TestHeaderHeight(t *testing.T) {
	want := strings.Count(duchessLogo, "\n") + 1
	if headerHeight != want {
		t.Errorf("headerHeight = %d, want %d (derived from duchessLogo line count)", headerHeight, want)
	}
	if headerHeight != 6 {
		t.Errorf("headerHeight = %d, want 6 (expected logo line count)", headerHeight)
	}
}

func TestRenderHeaderDimensions(t *testing.T) {
	m := rootModel{
		cfg: &config.Config{
			Profile:         "default",
			Region:          "us-east-1",
			RefreshInterval: 30,
		},
		width:  120,
		height: 40,
	}
	result := renderHeader(m, 120)
	gotH := lipgloss.Height(result)
	if gotH != headerHeight {
		t.Errorf("renderHeader height = %d, want %d (headerHeight)", gotH, headerHeight)
	}
	gotW := lipgloss.Width(result)
	if gotW != 120 {
		t.Errorf("renderHeader width = %d, want 120", gotW)
	}
}

func TestRenderHeaderZeroWidth(t *testing.T) {
	m := rootModel{width: 0, height: 24}
	result := renderHeader(m, 0)
	if result != "" {
		t.Errorf("renderHeader with width=0 should return \"\", got %q", result)
	}
}

func TestRenderHeaderContainsMetaLabels(t *testing.T) {
	m := rootModel{
		cfg: &config.Config{
			Profile:         "myprofile",
			Region:          "eu-west-1",
			RefreshInterval: 15,
		},
		width:  120,
		height: 40,
	}
	result := renderHeader(m, 120)
	// Strip ANSI sequences for text content check
	stripped := stripANSI(result)
	if !strings.Contains(stripped, "profile") {
		t.Error("renderHeader output should contain \"profile\" label")
	}
	if !strings.Contains(stripped, "region") {
		t.Error("renderHeader output should contain \"region\" label")
	}
}

func TestRenderHeaderVersionPresent(t *testing.T) {
	m := rootModel{
		cfg: &config.Config{
			Profile:         "default",
			Region:          "us-east-1",
			RefreshInterval: 30,
		},
		width:  120,
		height: 40,
	}
	result := renderHeader(m, 120)
	stripped := stripANSI(result)
	if !strings.Contains(stripped, "v"+version) {
		t.Errorf("renderHeader output should contain version string %q", "v"+version)
	}
}

