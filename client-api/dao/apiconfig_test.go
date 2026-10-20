package dao

import "testing"

func TestParseHallURL(t *testing.T) {
	cases := []struct {
		raw  string
		ip   string
		port string
		sip  string
	}{
		{"wss://gw.example.com:443", "wss://gw.example.com", "443", `["wss://gw.example.com:443"]`},
		{"ws://127.0.0.1:9000", "ws://127.0.0.1", "9000", `["ws://127.0.0.1:9000"]`},
		{"gw.example.com:8443", "wss://gw.example.com", "8443", `["wss://gw.example.com:8443"]`},
		{"https://127.0.0.1:1080/hall", "wss://127.0.0.1", "1080", `["wss://127.0.0.1:1080"]`},
	}
	for _, c := range cases {
		node := ParseHallURL(c.raw)
		if node == nil {
			t.Fatalf("parse %q got nil", c.raw)
		}
		if node.ServerIp != c.ip || node.ServerPort != c.port || node.Sip != c.sip {
			t.Fatalf("parse %q got %+v want ip=%s port=%s sip=%s", c.raw, node, c.ip, c.port, c.sip)
		}
	}
	if ParseHallURL("") != nil || ParseHallURL("   ") != nil {
		t.Fatal("empty hall url should be nil")
	}
}
