package gomod

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattermost/gobom/generators/gclient/deps"
	"github.com/mattermost/gobom/log"

	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/cyclonedx"
)

// Generator generates BOMs for Go modules projects
type Generator struct {
	gobom.BaseGenerator
}

func init() {
	gobom.RegisterGenerator(&Generator{})
}

// Configure sets the options for this Generator
func (g *Generator) Configure() error {
	return nil
}

// GenerateBOM returns a CycloneDX BOM for the specified package path
func (g *Generator) GenerateBOM(path string) (*cyclonedx.BOM, error) {
	file, err := os.Open(filepath.Join(path, "DEPS"))
	if err != nil {
		return nil, err
	}

	d, err := deps.New(file)

	d.SetTargetOS("linux", "mac", "windows")
	d.SetTargetCPU("x64", "x86", "arm64")
	d.SetHostOS("linux")
	d.SetHostCPU("x64")

	log.Debug("%v %v", d, err)

	err = d.Resolve()

	log.Debug("%v %v %v", d, err, d.Errors())

	for path, dep := range d.Deps() {
		if dep.Type() == deps.GitDepType {
			fmt.Printf("%s: %s (required by %s)\n", path, dep.(*deps.GitDep).URL, dep.(*deps.GitDep).Parent)
		} else {
			fmt.Printf("%s: CIPD dependency (required by %s)\n", path, dep.(*deps.CIPDDep).Parent)
		}
	}

	return &cyclonedx.BOM{
		Components: g.toComponents(d.Deps()),
	}, err
}

func (g *Generator) toComponents(deps map[string]deps.Dep) []*cyclonedx.Component {
	return nil
}
