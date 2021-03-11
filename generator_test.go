package gobom_test

import (
	"strings"
	"testing"

	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/cyclonedx"
)

type TestGenerator struct{}

func (*TestGenerator) Configure(gobom.Options) error {
	return nil
}

func (*TestGenerator) GenerateBOM(string) (*cyclonedx.BOM, error) {
	return nil, nil
}

func TestResolveName(t *testing.T) {
	g := &TestGenerator{}

	name := gobom.ResolveShortName(g)
	if name != "gobom_test" {
		t.Errorf("bad generator name: expected 'gobom_test', observed '%s'", name)
	}
}

func TestRegisterGenerator(t *testing.T) {
	g := &TestGenerator{}
	registered := false

	gobom.OnGeneratorRegistered(func(key string, g2 gobom.Generator) {
		if strings.HasSuffix(key, "/gobom_test") && g2 == g {
			if registered == true {
				t.Error("OnGeneratorRegistered callback executed multiple times")
			}
			registered = true
		} else {
			t.Errorf("OnGeneratorRegistered callback executed for unexpected generator: '%s'", g2)
		}
	})

	gobom.RegisterGenerator(g)

	if registered == false {
		t.Errorf("OnGeneratorRegistered callback not executed")
	}

	generators := gobom.Generators()
	if len(generators) != 1 {
		t.Errorf("expected exactly 1 generator to be registered, saw %d", len(generators))
	}

	g2, err := gobom.GetGenerator(gobom.ResolveShortName(g))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if g2 != g {
		t.Error("output from GetGenerator doesn't match expected generator")
	}

	g2, err = gobom.GetGenerator(gobom.ResolveName(g))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if g2 != g {
		t.Error("output from GetGenerator doesn't match expected generator")
	}
}

func TestGetGenerator(t *testing.T) {
	g, err := gobom.GetGenerator("nosuchthing")
	if err == nil {
		t.Error("expected an error when getting an inexistent generator, saw nil")
	}
	if g != nil {
		t.Errorf("expected nil generator, saw %v", g)
	}

	g, err = gobom.GetGenerator("packagename/nosuchthing")
	if err == nil {
		t.Error("expected an error when getting an inexistent generator, saw nil")
	}
	if g != nil {
		t.Errorf("expected nil generator, saw %v", g)
	}
}
