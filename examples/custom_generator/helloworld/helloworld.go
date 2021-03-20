package helloworld

import (
	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/cyclonedx"
	"github.com/mattermost/gobom/log"
)

// Generator implements our custom helloworld BOM generator
type Generator struct {
	gobom.BaseGenerator

	// Recurse is a global property defined in BaseGenerator;
	// this overrides the help text in `gobom help custom_generator/helloworld`
	Recurse string `gobom:",,NOTE: helloworld does not recurse"`

	// Panic is a custom global property exposed as a command-line flag
	Panic bool `gobom:"panic,X,crash the tool during configure"`

	// HelloworldGreeting is a locally-scoped custom property
	HelloworldGreeting string `gobom:"sets the greeting to use; defaults to 'Hello'"`
}

// Configure configures the Generator
func (g *Generator) Configure() error {
	if g.HelloworldGreeting == "" {
		g.HelloworldGreeting = "Hello"
	}
	if g.Panic {
		panic("Goodbye world!")
	}
	return nil
}

// GenerateBOM generates a bill of materials for a specified path.
// In our example, it just logs a greeting.
func (g *Generator) GenerateBOM(string) (*cyclonedx.BOM, error) {
	log.Warn("%s world!", g.HelloworldGreeting)
	return nil, nil
}

func init() {
	gobom.RegisterGenerator(&Generator{})
}
