package gobom

import "testing"

func TestPURL(t *testing.T) {
	purl := PURL("foo", "bar/baz", "quux")
	if purl != "pkg:foo/bar/baz@quux" {
		t.Fatalf("unexpected PURL output: '%s'", purl)
	}
}
