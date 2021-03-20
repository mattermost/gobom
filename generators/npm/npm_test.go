package npm

import (
	"testing"
)

func TestGenerateBOM(t *testing.T) {
	g := Generator{}
	g.Recurse = true
	g.Configure()

	bom, err := g.GenerateBOM("./testdata/testpackage")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := 0
	expected := map[string]string{
		"testpackage":   "pkg:npm/testpackage@1.0.0",
		"js-tokens":     "pkg:npm/js-tokens@4.0.0",
		"loose-envify":  "pkg:npm/loose-envify@1.4.0",
		"object-assign": "pkg:npm/object-assign@4.1.1",
		"react":         "pkg:npm/react@17.0.1",
	}
	for _, component := range bom.Components {
		if purl, ok := expected[component.Name]; ok && purl == component.PURL {
			found++
		} else if ok && purl != component.PURL {
			t.Errorf("unexpected purl: expected '%s', saw '%s'", purl, component.PURL)
		} else {
			t.Errorf("unexpected component: '%s'", component.Name)
		}
	}
	if found != len(expected) {
		t.Errorf("expected %d components, saw %d", len(expected), found)
	}
}
