package runtime

import (
	"os"
	"strings"
	"testing"
)

func TestPrependNodeBinDirRewritesExistingPath(t *testing.T) {
	base := []string{"HOME=/home/x", "PATH=/usr/bin:/bin"}
	got := prependNodeBinDir(base, "/opt/node/bin")
	want := "PATH=/opt/node/bin" + string(os.PathListSeparator) + "/usr/bin:/bin"
	if got[1] != want {
		t.Fatalf("PATH entry = %q, want %q", got[1], want)
	}
}

func TestPrependNodeBinDirAddsPathWhenAbsent(t *testing.T) {
	base := []string{"HOME=/home/x"}
	got := prependNodeBinDir(base, "/opt/node/bin")
	if len(got) != 2 || got[1] != "PATH=/opt/node/bin" {
		t.Fatalf("got %v, want PATH appended", got)
	}
}

func TestPrependNodeBinDirNoopOnEmptyDir(t *testing.T) {
	base := []string{"PATH=/usr/bin"}
	got := prependNodeBinDir(base, "")
	if strings.Join(got, ",") != strings.Join(base, ",") {
		t.Fatalf("expected no change, got %v", got)
	}
}
