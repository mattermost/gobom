package cocoapods

import (
	"testing"
)

func TestGenerateBOM(t *testing.T) {
	g := Generator{}
	g.Recurse = true
	g.Configure()

	bom, err := g.GenerateBOM("./testdata/testapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := 0
	expected := map[string]string{
		"AFNetworking/Reachability":    "pkg:cocoapods/AFNetworking/Reachability@2.7.0",
		"AFNetworking/Serialization":   "pkg:cocoapods/AFNetworking/Serialization@2.7.0",
		"AFNetworking/UIKit":           "pkg:cocoapods/AFNetworking/UIKit@2.7.0",
		"ORStackView":                  "pkg:cocoapods/ORStackView@3.0.1",
		"AFNetworking/Security":        "pkg:cocoapods/AFNetworking/Security@2.7.0",
		"FLKAutoLayout":                "pkg:cocoapods/FLKAutoLayout@0.2.1",
		"./testdata/testapp":           "pkg:generic/./testdata/testapp@unknown",
		"AFNetworking":                 "pkg:cocoapods/AFNetworking@2.7.0",
		"AFNetworking/NSURLConnection": "pkg:cocoapods/AFNetworking/NSURLConnection@2.7.0",
		"AFNetworking/NSURLSession":    "pkg:cocoapods/AFNetworking/NSURLSession@2.7.0",
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
