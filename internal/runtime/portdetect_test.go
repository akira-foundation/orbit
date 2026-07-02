package runtime

import "testing"

func TestExtractPortIgnoresConflictLines(t *testing.T) {
	conflictLines := []string{
		"Port 5173 is in use, trying another one...",
		"Port 5174 is in use, trying another one...",
		"error: listen EADDRINUSE: address already in use :::3000",
		"⚠️  Port 3000 is already in use",
	}
	for _, line := range conflictLines {
		if port, _ := extractPortFromLine(line); port != 0 {
			t.Fatalf("conflict line yielded port %d: %q", port, line)
		}
	}
}

func TestExtractPortHandlesANSIColoredOutput(t *testing.T) {
	line := "  \x1b[32m➜\x1b[39m  \x1b[1mLocal\x1b[22m:   \x1b[36mhttp://localhost:\x1b[1m5177\x1b[22m/\x1b[39m"
	port, strong := extractPortFromLine(line)
	if port != 5177 || !strong {
		t.Fatalf("ansi line = (%d,%v) want (5177,true)", port, strong)
	}
}

func TestExtractPortStillDetectsRealPorts(t *testing.T) {
	cases := []struct {
		line   string
		port   int
		strong bool
	}{
		{"  ➜  Local:   http://localhost:5176/", 5176, true},
		{"ready - started server on 0.0.0.0:3000", 3000, true},
		{"Server listening at port: 4321", 4321, false},
	}
	for _, c := range cases {
		port, strong := extractPortFromLine(c.line)
		if port != c.port || strong != c.strong {
			t.Fatalf("extract(%q) = (%d,%v) want (%d,%v)", c.line, port, strong, c.port, c.strong)
		}
	}
}
