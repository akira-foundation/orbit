package share

import "testing"

func TestShareURL(t *testing.T) {
	got := ShareURL("192.168.1.42", "2443", "shop")
	if got != "https://192.168.1.42:2443/?__orbit=shop" {
		t.Fatalf("ShareURL = %q", got)
	}
}

func TestPrimaryLANIPReturnsNonLoopback(t *testing.T) {
	ip, err := PrimaryLANIP()
	if err != nil {
		t.Skipf("no LAN interface in test env: %v", err)
	}
	if ip == "" || ip == "127.0.0.1" {
		t.Fatalf("PrimaryLANIP = %q, want non-loopback", ip)
	}
}
