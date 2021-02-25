package gobom

import (
	"path"
	"reflect"

	"github.com/mattermost/gobom/cyclonedx"
)

// Generator is a set of methods for generating CycloneDX BOMs
type Generator interface {
	Configure(Options) error
	GenerateBOM(string) (*cyclonedx.BOM, error)
}

// Options controls various configurable aspects of BOM generation
type Options struct {
	IncludeSubcomponents bool
	IncludeTests         bool
	Recurse              bool
	Properties           map[string]string
}

var generators = make(map[string]Generator)

// RegisterGenerator registers a Generator for use by gobom.
// Returns true if the registration replaced an existing Generator.
// Generators of different type are identified by their package
// name and only one Generator of a specific type can be registered
// at a time.
func RegisterGenerator(g Generator) bool {
	key := path.Base(reflect.ValueOf(g).Elem().Type().PkgPath())
	_, exists := generators[key]
	generators[key] = g
	return exists
}

// Generators returns the currently registered Generators as a
// map of generator type name to generator instance
func Generators() map[string]Generator {
	out := make(map[string]Generator)
	for key, generator := range generators {
		out[key] = generator
	}
	return out
}
