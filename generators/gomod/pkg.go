package gomod

import (
	"fmt"
	"strings"

	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/cyclonedx"
)

type module struct {
	Path    string
	Version string
	Replace *module
}

type pkg struct {
	ImportPath string
	Name       string
	ForTest    string
	Module     *module
	Standard   bool
}

func (p pkg) toComponents() []*cyclonedx.Component {
	components := []*cyclonedx.Component{&cyclonedx.Component{
		Type:        cyclonedx.Library,
		Name:        p.ImportPath,
		Description: "Golang package\n\n",
	}}
	if p.Module != nil {
		components[0].Group = p.Module.Path
		components[0].Version = normalizeVersion(p.Module.Version)
		if p.Module.Replace != nil {
			// Include replacements as additional Components.
			// Rationale: A replacement is in most cases a fork poorly
			// documented in vulnerability databases; we're much more
			// likely to find matches for the original. But it could
			// also be a widely-known drop-in replacement. Incuding
			// both the original and the replacement avoids false
			// negatives.
			p.Module = p.Module.Replace
			components = append(components, p.toComponents()...)
			components[1].Description = fmt.Sprintf("%sReplaces package in module %s\n", components[1].Description, components[0].Group)
			components[0].Description = fmt.Sprintf("%sReplaced with package in module %s\n", components[0].Description, components[1].Group)
		}
	} else {
		if p.Standard {
			components[0].Group = "github.com/golang/go"
		} else if p.ForTest != "" || (p.Name == "main" && strings.HasSuffix(p.ImportPath, ".test")) {
			// this is a test package or a generated test main from the build cache; ignore
			return []*cyclonedx.Component{}
		}
		components[0].Version = "unknown"
	}
	components[0].PURL = gobom.PURL(gobom.GolangPackage, components[0].Name, components[0].Version)
	return components
}
