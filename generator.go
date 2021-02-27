package gobom

import (
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/mattermost/gobom/cyclonedx"
)

// Generator is a set of methods for generating CycloneDX BOMs
type Generator interface {
	Configure(Options) error
	GenerateBOM(string) (*cyclonedx.BOM, error)
}

// ResolveName returns the full name of the Generator,
// consisting of the package name and the one path
// component above it, e.g. "generators/cocoapods".
//
// The name is unique among registered Generators.
func ResolveName(g Generator) string {
	pkg := reflect.ValueOf(g).Elem().Type().PkgPath()
	return fmt.Sprintf("%s/%s", path.Base(path.Dir(pkg)), path.Base(pkg))
}

// ResolveShortName returns the short name of the Generator,
// consisting only of the package name, e.g. "cocoapods".
//
// Short names are not guaranteed to be unique.
func ResolveShortName(g Generator) string {
	return path.Base(ResolveName(g))
}

// Options controls various configurable aspects of BOM generation
type Options struct {
	IncludeSubcomponents bool
	IncludeTests         bool
	Recurse              bool
}

var registerCallbacks = []func(key string, g Generator){}

var generators = make(map[string]Generator)

// RegisterGenerator registers a Generator for use by gobom.
// Returns true if the registration replaced an existing Generator.
// Generators of different type are identified by their package
// path and only one Generator of a specific type can be registered
// at a time.
func RegisterGenerator(g Generator) bool {
	key := ResolveName(g)
	_, exists := generators[key]
	generators[key] = g
	for _, cb := range registerCallbacks {
		cb(key, g)
	}
	return exists
}

// OnGeneratorRegistered registers a function to be called when a new Generator is added
func OnGeneratorRegistered(callback func(key string, g Generator)) {
	registerCallbacks = append(registerCallbacks, callback)
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

// GetGenerator returns the Generator corresponding to a specified name.
// Both short names and full names are accepted. If an ambiguous short name
// is specified, GetGenerator returns an error.
func GetGenerator(name string) (Generator, error) {
	if strings.Contains(name, "/") {
		// full name
		g, ok := generators[name]
		if !ok {
			return nil, fmt.Errorf("no such generator: '%s'", name)
		}
		return g, nil
	}

	// short name
	var g Generator
	for key, value := range generators {
		if path.Base(key) == name {
			if g == nil {
				g = value
			} else {
				return nil, fmt.Errorf("ambiguous generator name: '%s'", name)
			}
		}
	}
	if g == nil {
		return nil, fmt.Errorf("no such generator: '%s'", name)
	}
	return g, nil
}
