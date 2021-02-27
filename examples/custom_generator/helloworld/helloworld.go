package helloworld

import (
	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/cyclonedx"
	"github.com/mattermost/gobom/log"
)

// Generator implements our custom helloworld BOM generator
type Generator struct {
	HelloworldGreeting string `gobom:"sets the greeting to use; defaults to 'Hello'"`
}

// Configure configures the Generator
func (g *Generator) Configure(gobom.Options) error {
	if g.HelloworldGreeting == "" {
		g.HelloworldGreeting = "Hello"
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
