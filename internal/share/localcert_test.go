package share

import "testing"

func TestLocalIPHost(t *testing.T) {
	got := LocalIPHost("192.168.178.26")
	if got != "192-168-178-26.local-ip.sh" {
		t.Fatalf("LocalIPHost = %q", got)
	}
}
