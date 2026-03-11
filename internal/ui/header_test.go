package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/jacobscunn07/duchess/internal/config"
)

func TestHeaderHeight(t *testing.T) {
	wantLogoHeight := strings.Count(duchessLogo, "\n") + 1
	if logoHeight != wantLogoHeight {
		t.Errorf("logoHeight = %d, want %d (derived from duchessLogo line count)", logoHeight, wantLogoHeight)
	}
	if logoHeight != 6 {
		t.Errorf("logoHeight = %d, want 6 (expected logo line count)", logoHeight)
	}
	// headerHeight = logoHeight + 1 (extra row for centered version below logo)
	if headerHeight != logoHeight+1 {
		t.Errorf("headerHeight = %d, want logoHeight+1 = %d", headerHeight, logoHeight+1)
	}
	if headerHeight != 7 {
		t.Errorf("headerHeight = %d, want 7 (6 logo rows + 1 version row)", headerHeight)
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

