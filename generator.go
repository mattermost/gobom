package gobom

import (
	"github.com/mattermost/gobom/cyclonedx"
)

// Generator is a set of methods for generating CycloneDX BOMs
type Generator interface {
	Configure(Options) error
	GenerateBOM(string) (*cyclonedx.BOM, error)
	Name() string
}

// Options controls various configurable aspects of BOM generation
type Options struct {
	IncludeSubcomponents bool
	IncludeTests         bool
	Recurse              bool
	Properties           map[string]string
}
