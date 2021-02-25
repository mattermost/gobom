package gomod

import (
	"testing"

	"github.com/mattermost/gobom"
)

func TestGenerateBom(t *testing.T) {
	generator := Generator{}

	generator.Configure(gobom.Options{
		IncludeSubcomponents: true,
		IncludeTests:         true,
		Recurse:              true,
	})
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
	if bom.Components[0].Components[0].Name != "github.com/mattermost/gobom/gomod/testdata/testpackage" {
		t.Fatalf("unexpected package name '%s'", bom.Components[0].Name)
	}

	generator.Configure(gobom.Options{
		IncludeSubcomponents: false,
		IncludeTests:         true,
		Recurse:              true,
	})
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
}
