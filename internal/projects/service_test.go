package projects

import "testing"

func TestSlugifyPreservesDots(t *testing.T) {
	got := Slugify("nosferry.com")
	if got != "nosferry.com" {
		t.Fatalf("Slugify(%q) = %q, want nosferry.com", "nosferry.com", got)
	}
}

func TestSlugifyStripsVendorPrefixAndCollapsesJunk(t *testing.T) {
	got := Slugify("nosferry/nosferry.com")
	if got != "nosferry-nosferry.com" {
		t.Fatalf("Slugify(%q) = %q, want nosferry-nosferry.com", "nosferry/nosferry.com", got)
	}
}

func TestSlugifyEmptyFallsBackToProject(t *testing.T) {
	if got := Slugify("   "); got != "project" {
		t.Fatalf("Slugify(blank) = %q, want project", got)
	}
}
