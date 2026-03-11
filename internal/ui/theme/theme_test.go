package theme

import (
	"testing"
)

func TestDarkTheme_AccentTrueColor(t *testing.T) {
	th := DarkTheme()
	if th.Accent.TrueColor != "#FF9900" {
		t.Errorf("expected Accent.TrueColor to be #FF9900, got %s", th.Accent.TrueColor)
	}
}

func TestDarkTheme_BackgroundTrueColor(t *testing.T) {
	th := DarkTheme()
	if th.Background.TrueColor != "#232F3E" {
		t.Errorf("expected Background.TrueColor to be #232F3E, got %s", th.Background.TrueColor)
	}
}

func TestDarkTheme_SurfaceTrueColor(t *testing.T) {
	th := DarkTheme()
	if th.Surface.TrueColor != "#2D2D2D" {
		t.Errorf("expected Surface.TrueColor to be #2D2D2D, got %s", th.Surface.TrueColor)
	}
}

func TestDarkTheme_AccentANSIFallbacks(t *testing.T) {
	th := DarkTheme()
	if th.Accent.ANSI256 != "214" {
		t.Errorf("expected Accent.ANSI256 to be 214, got %s", th.Accent.ANSI256)
	}
	if th.Accent.ANSI != "3" {
		t.Errorf("expected Accent.ANSI to be 3, got %s", th.Accent.ANSI)
	}
}

func TestDefaultTheme_BackgroundTrueColor(t *testing.T) {
	if DefaultTheme.Background.TrueColor != "#232F3E" {
		t.Errorf("expected DefaultTheme.Background.TrueColor to be #232F3E, got %s", DefaultTheme.Background.TrueColor)
	}
}
