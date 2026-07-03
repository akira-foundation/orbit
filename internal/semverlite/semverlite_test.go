package semverlite

import "testing"

func TestResolve(t *testing.T) {
	available := []string{"24.18.0", "22.23.1", "20.20.2"}

	cases := []struct {
		constraint string
		want       string
		wantOK     bool
	}{
		{"^22.0.0", "22.23.1", true},
		{"^20.0.0", "20.20.2", true},
		{"~20.20.0", "20.20.2", true},
		{"~20.19.0", "", false},
		{">=20.0.0", "24.18.0", true},
		{">=20.0.0 <24.0.0", "22.23.1", true},
		{"20.x", "20.20.2", true},
		{"20", "20.20.2", true},
		{"20.20.2", "20.20.2", true},
		{"20.20.3", "", false},
		{"^18.0.0|^22.0.0", "22.23.1", true},
		{"garbage-constraint", "", false},
		{"", "", false},
	}

	for _, c := range cases {
		got, ok := Resolve(c.constraint, available)
		if ok != c.wantOK || got != c.want {
			t.Errorf("Resolve(%q) = (%q, %v), want (%q, %v)", c.constraint, got, ok, c.want, c.wantOK)
		}
	}
}

func TestParseVersion(t *testing.T) {
	cases := []struct {
		in     string
		want   Version
		wantOK bool
	}{
		{"8.3.17", Version{8, 3, 17}, true},
		{"v8.3.17", Version{8, 3, 17}, true},
		{"8.3", Version{8, 3, 0}, true},
		{"8", Version{8, 0, 0}, true},
		{"not-a-version", Version{}, false},
	}
	for _, c := range cases {
		got, ok := ParseVersion(c.in)
		if ok != c.wantOK || got != c.want {
			t.Errorf("ParseVersion(%q) = (%+v, %v), want (%+v, %v)", c.in, got, ok, c.want, c.wantOK)
		}
	}
}
