package updater

import "testing"

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		remote   string
		current  string
		expected bool
	}{
		{"v0.1.3", "0.1.0", true},
		{"v0.2.0", "0.1.3", true},
		{"v0.2.0", "0.2.0", false},
		{"v0.1.3", "0.2.0", false},
		{"v1.0.0", "0.9.9", true},
		{"v0.2.1", "0.2.0", true},
		{"v0.2.0", "0.2.0+dirty", false},
		{"v0.2.1", "0.2.0+dirty", true},
	}

	for _, tc := range tests {
		got := isNewerVersion(tc.remote, tc.current)
		if got != tc.expected {
			t.Errorf("isNewerVersion(%q, %q) = %v; want %v", tc.remote, tc.current, got, tc.expected)
		}
	}
}
