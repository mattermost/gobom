package gomod

import (
	"testing"
)

func TestGenerateBom(t *testing.T) {
	generator := Generator{}

	generator.Recurse = true
	generator.GomodPackages = true
	generator.GomodTests = false
	generator.Configure()
	if !generator.GomodPackages {
		t.Fatal("GomodPackages should be true after Configure")
	}
	if generator.GomodTests {
		t.Fatal("GomodTests should be false after Configure")
	}

	bom, err := generator.GenerateBOM("./testdata/testpackage")
	if err != nil {
		t.Fatalf("GenerateBOM failed: %v", err)
	}
	if len(bom.Components) != 1 {
		t.Fatal("BOM should contain exactly one Component")
	}
	if len(bom.Components[0].Components) != 1 {
		t.Fatal("Component should contain exactly one subcomponent")
	}
	if bom.Components[0].Components[0].Name != "github.com/mattermost/gobom/generators/gomod/testdata/testpackage" {
		t.Fatalf("unexpected package name '%s'", bom.Components[0].Components[0].Name)
	}

	generator.Recurse = true
	generator.GomodPackages = false
	generator.GomodTests = false
	generator.Configure()
	bom, err = generator.GenerateBOM("./testdata/testpackage")
	if err != nil {
		t.Fatalf("GenerateBOM failed: %v", err)
	}
	if len(bom.Components) != 1 {
		t.Fatal("BOM should contain exactly one Component")
	}
	if len(bom.Components[0].Components) != 0 {
		t.Fatal("Component should not contain any subcomponents")
	}
	if bom.Components[0].Name != "github.com/mattermost/gobom" {
		t.Fatalf("unexpected module name '%s'", bom.Components[0].Name)
	}

	generator.Recurse = true
	generator.GomodPackages = false
	generator.GomodTests = true
	generator.Configure()
	bom, err = generator.GenerateBOM("./testdata/testpackage")
	if err != nil {
		t.Fatalf("GenerateBOM failed: %v", err)
	}
	if len(bom.Components) != 2 {
		t.Fatalf("BOM should contain exactly 2 Components, saw %d", len(bom.Components))
	}
	if (bom.Components[0].Name != "github.com/mattermost/gobom" || bom.Components[1].Name != "github.com/golang/go") &&
		(bom.Components[1].Name != "github.com/mattermost/gobom" || bom.Components[0].Name != "github.com/golang/go") {
		t.Fatalf("unexpected module names in BOM")
	}

	generator.Recurse = true
	generator.Filters = []string{"release"}
	generator.GomodPackages = false
	generator.GomodTests = true
	generator.Configure()
	if generator.GomodTests {
		t.Fatal("GomodTests should be false after Configure")
	}

	generator.Recurse = true
	generator.Filters = []string{"release", "test"}
	generator.GomodPackages = false
	generator.GomodTests = true
	generator.Configure()
	if !generator.GomodTests {
		t.Fatal("GomodTests should be true after Configure")
	}
}
