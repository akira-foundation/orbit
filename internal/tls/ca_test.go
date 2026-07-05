package tls

import "testing"

func withTestTLSDir(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestEnsureLeafCoversMultiLabelDomain(t *testing.T) {
	withTestTLSDir(t)

	mat, err := Ensure("orbit.test", []string{"nosferry.com.orbit.test"})
	if err != nil {
		t.Fatal(err)
	}
	cert, err := readCert(mat.LeafCert)
	if err != nil {
		t.Fatal(err)
	}
	if err := cert.VerifyHostname("nosferry.com.orbit.test"); err != nil {
		t.Fatalf("expected leaf to cover the dotted domain: %v", err)
	}
}

func TestEnsureRegeneratesWhenNewDomainNotCovered(t *testing.T) {
	withTestTLSDir(t)

	if _, err := Ensure("orbit.test", nil); err != nil {
		t.Fatal(err)
	}

	mat, err := Ensure("orbit.test", []string{"app.two.orbit.test"})
	if err != nil {
		t.Fatal(err)
	}
	cert, err := readCert(mat.LeafCert)
	if err != nil {
		t.Fatal(err)
	}
	if err := cert.VerifyHostname("app.two.orbit.test"); err != nil {
		t.Fatalf("expected regenerated leaf to cover the new domain: %v", err)
	}
}

func TestLeafCoversDomainsEmptyListIsAlwaysTrue(t *testing.T) {
	if !leafCoversDomains("/does/not/exist.pem", nil) {
		t.Fatal("expected true for an empty domain list regardless of file state")
	}
}

func TestDedupeDomainsRemovesBlankAndDuplicates(t *testing.T) {
	got := dedupeDomains([]string{"a", "", "b", "a"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %v, want [a b]", got)
	}
}
