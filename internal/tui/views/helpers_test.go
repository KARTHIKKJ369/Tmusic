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

func TestDeleteWord(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"   ", ""},
		{"hello", ""},
		{"hello ", ""},
		{"hello world", "hello "},
		{"hello world   ", "hello "},
		{"one two three", "one two "},
		{"linkin park - numb", "linkin park - "},
		{"linkin park - ", "linkin park "},
		{"linkin park ", "linkin "},
		{"linkin ", ""},
		{"hello-world", "hello-"},
		{"hello-", "hello"},
		{"café au lait", "café au "},
	}

	for _, tt := range tests {
		got := DeleteWord(tt.input)
		if got != tt.expected {
			t.Errorf("DeleteWord(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}
