package share

import (
	"strings"
	"testing"
)

func TestQRDataURI(t *testing.T) {
	uri, err := QRDataURI("http://192.168.1.42:2080/?__orbit=shop")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(uri, "data:image/png;base64,") {
		t.Fatalf("prefix = %q", uri[:30])
	}
	if len(uri) < 200 {
		t.Fatalf("uri too short: %d", len(uri))
	}
}
