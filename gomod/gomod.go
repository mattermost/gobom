package gomod

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/cyclonedx"
)

const (
	goStdlibModule = "github.com/golang/go"
	unknownVersion = "unknown"
)

// Generator generates BOMs for Go modules projects
type Generator struct {
	options gobom.Options
}

// Name returns the name of the Generator
func (g *Generator) Name() string {
	return "Go modules generator"
}

// Configure sets the options for this Generator
func (g *Generator) Configure(options gobom.Options) error {
	g.options = options
	return nil
}

// GenerateBOM returns a CycloneDX BOM for the specified package path
// TODO: should follow symlinks in recursive mode?
func (g *Generator) GenerateBOM(path string) (*cyclonedx.BOM, error) {
	packages, err := g.listPackages(path)
	if err != nil {
		return nil, err
	}
	modules := mapPackagesToModules(packages)
	if err = resolveWhy(path, modules); err != nil {
		return nil, err
	}
	if err = resolveGoVersion(modules); err != nil {
		return nil, err
	}

	if !g.options.IncludeSubcomponents {
		for _, module := range modules {
			module.Components = nil
		}
	}

	return &cyclonedx.BOM{Components: modules}, nil
}

func (g *Generator) listPackages(path string) ([]*cyclonedx.Component, error) {
	packages := make([]*cyclonedx.Component, 0)

	args := []string{"list", "-mod", "readonly", "-deps", "-json"}
	if g.options.IncludeTests {
		args = append(args, "-test")
	}
	if g.options.Recurse {
		args = append(args, "./...")
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = path
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("go list: %v", err)
	}
	cmd.Start()
	decoder := json.NewDecoder(stdout)
	for decoder.More() {
		pkg := &pkg{}
		err := decoder.Decode(pkg)
		if err != nil {
			return nil, fmt.Errorf("go list: %v", err)
		}
		packages = append(packages, pkg.toComponents()...)
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("go list: %v", err)
	}
	return packages, nil
}

func resolveGoVersion(modules []*cyclonedx.Component) error {
	cmd := exec.Command("go", "version")
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	var major, minor, patch = 0, 0, 0
	n, err := fmt.Sscanf(string(out), "go version go%d.%d.%d", &major, &minor, &patch)
	if n == 0 {
		return err
	}
	version := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	for _, module := range modules {
		if module.Name == goStdlibModule && module.Version == unknownVersion {
			module.Version = version
			module.PURL = gobom.PURL(gobom.GolangPackage, module.Name, version)
		}
	}

	return nil
}

func resolveWhy(path string, modules []*cyclonedx.Component) error {
	cmd := exec.Command("go", "mod", "why", "-m", "all")
	cmd.Dir = path
	why, err := cmd.Output()
	if err != nil {
		return err
	}

	for _, module := range modules {
		// find this module in the `go mod why` output
		start := bytes.Index(why, []byte(fmt.Sprintf("# %s\n", module.Name)))
		if start == -1 {
			continue
		}
		end := start + bytes.Index(why[start:], []byte("\n#"))
		if end < start {
			end = len(why)
		}
		// build description
		lines := strings.Split(string(why[start:end]), "\n")
		module.Description = fmt.Sprintf("%s\nRequired by:\n", module.Description)
		for j := len(lines) - 1; j >= 0; j-- {
			if lines[j] != "" && !strings.HasPrefix(lines[j], "#") {
				module.Description = fmt.Sprintf("%s\t%s\n", module.Description, lines[j])
			}
		}
	}

	return nil
}

func mapPackagesToModules(packages []*cyclonedx.Component) []*cyclonedx.Component {
	moduleMap := make(map[string]*cyclonedx.Component)
	for _, pkg := range packages {
		module, exists := moduleMap[pkg.Group]
		if !exists {
			module = &cyclonedx.Component{
				Type:        cyclonedx.Library,
				Name:        pkg.Group,
				Version:     pkg.Version,
				Description: fmt.Sprintf("Golang module\n\nPackages:\n"),
				PURL:        gobom.PURL(gobom.GolangPackage, pkg.Group, pkg.Version),
			}
		}
		module.Components = append(module.Components, pkg)
		if len(module.Components) < 5 {
			module.Description = fmt.Sprintf("%s\t%s\n", module.Description, pkg.Name)
		} else {
			lastLine := strings.LastIndex(module.Description, "\t")
			module.Description = fmt.Sprintf("%s\t... and %d more\n", module.Description[0:lastLine], len(module.Components)-3)
		}

		moduleMap[pkg.Group] = module
	}

	modules := make([]*cyclonedx.Component, 0, len(moduleMap))
	for _, module := range moduleMap {
		modules = append(modules, module)
	}

	return modules
}

func normalizeVersion(version string) string {
	if version == "" {
		return unknownVersion
	} else if strings.HasPrefix(version, "v") {
		return version[1:]
	}
	return version
}
