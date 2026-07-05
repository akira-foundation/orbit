package runtime

import "testing"

func TestStripEnvVarRemovesCIEntirely(t *testing.T) {
	base := []string{"HOME=/home/x", "CI=true", "PATH=/usr/bin"}
	got := stripEnvVar(base, "CI")
	for _, kv := range got {
		if len(kv) >= 3 && kv[:3] == "CI=" {
			t.Fatalf("expected CI to be fully absent, found %q in %v", kv, got)
		}
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 remaining entries, got %v", got)
	}
}

func TestStripEnvVarNoopWhenAbsent(t *testing.T) {
	base := []string{"HOME=/home/x", "PATH=/usr/bin"}
	got := stripEnvVar(base, "CI")
	if len(got) != 2 || got[0] != base[0] || got[1] != base[1] {
		t.Fatalf("expected no change, got %v", got)
	}
}
