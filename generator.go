package gobom

import (
	"fmt"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/mattermost/gobom/cyclonedx"
)

// Generator is a set of methods for generating CycloneDX BOMs
type Generator interface {
	Configure() error
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

// VisitProperties visits all the gobom properites in a Generator
func VisitProperties(g Generator, visitor func(reflect.StructField, reflect.Value)) {
	visitProperties(g, visitor, nil)
}

func visitProperties(g interface{}, visitor func(reflect.StructField, reflect.Value), exclude map[string]bool) {
	e := reflect.ValueOf(g).Elem()
	t := e.Type()
	if t.Kind() != reflect.Struct {
		return
	}
	if exclude == nil {
		exclude = make(map[string]bool)
	}
	anons := []int{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag
		if _, ok := tag.Lookup("gobom"); ok && !exclude[field.Name] {
			visitor(field, e.Field(i))
			exclude[field.Name] = true
		} else if field.Anonymous {
			anons = append(anons, i)
		}
	}
	for _, i := range anons {
		visitProperties(e.Field(i).Addr().Interface(), visitor, exclude)
	}
}

// BaseGenerator implements a no-op Generator
type BaseGenerator struct {
	// Recurse tells a Generator to run in recursive mode,
	// walking all subdirectories and generating components
	// for every applicable path it sees.
	Recurse bool `gobom:"recurse,r,scan the target path recursively"`

	// Excludes tells a Generator to exclude the specified
	// paths when running in recursive mode. This is a global
	// option; a Generator may also expose a scoped excludes
	// option as a property. If both are specified, both should
	// be applied.
	Excludes *regexp.Regexp `gobom:"excludes,x,regexp of paths to exclude in recursive mode"`

	// Filters specifies the filtering presets to pass to the
	// Generator.
	//
	// Default presets include 'release' and 'test',
	// and should be implemented by all Generators. The 'release'
	// preset should configure the Generator to include only
	// production dependencies in its output; the 'test' preset
	// should configure the output to include only non-production
	// dependencies.
	Filters []string `gobom:"filters,f,filtering presets to pass to generators, e.g. 'release' or 'test'"`
}

// Configure initializes the Generator
func (*BaseGenerator) Configure() error {
	return nil
}

// GenerateBOM returns a CycloneDX BOM for the specified package path
func (*BaseGenerator) GenerateBOM(string) (*cyclonedx.BOM, error) {
	panic("not implemented")
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
