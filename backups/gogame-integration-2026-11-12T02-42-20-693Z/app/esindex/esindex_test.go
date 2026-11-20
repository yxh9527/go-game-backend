package esindex

import "testing"

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/fg/config/pool/bxjg", "/fg/config/pool/bxjg"},
		{"/config/pool/bxjg", "/fg/config/pool/bxjg"},
		{"config/pool/bxjg", "/fg/config/pool/bxjg"},
		{"/agent/1/pool/bxjg", "/fg/agent/1/pool/bxjg"},
		{"/fg/agent/1/pool/bxjg", "/fg/agent/1/pool/bxjg"},
		{"", ""},
		{"/other/pool/bxjg", ""},
	}
	for _, tt := range tests {
		if got := NormalizeKey(tt.in); got != tt.want {
			t.Fatalf("NormalizeKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
