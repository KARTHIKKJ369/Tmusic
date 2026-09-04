package views

import (
	"strings"
	"testing"

	"github.com/KARTHIKKJ369/Tmusic/internal/tui/styles"
)

func TestTruncANSI(t *testing.T) {
	colored := styles.Primary.Render("Hello World This Is A Long Styled String")
	truncated := Trunc(colored, 10)
	if strings.HasSuffix(truncated, "K") {
		t.Errorf("Trunc produced trailing raw 'K': %q", truncated)
	}
	if VisibleLen(truncated) > 10 {
		t.Errorf("expected visual width <= 10, got %d", VisibleLen(truncated))
	}
}

func TestPadANSI(t *testing.T) {
	colored := styles.Primary.Render("Short")
	padded := Pad(colored, 20)
	if VisibleLen(padded) != 20 {
		t.Errorf("expected visual width 20, got %d", VisibleLen(padded))
	}
}
