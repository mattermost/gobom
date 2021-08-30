package gomod

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/cyclonedx"
	"github.com/mattermost/gobom/log"
)

// Generator generates BOMs for Go modules projects
type Generator struct {
	gobom.BaseGenerator

	GomodTests    bool           `gobom:"set to false to exclude test-only dependencies; defaults to true"`
	GomodPackages bool           `gobom:"set to true to include packages as subcomponents in the BOM"`
	GomodMainOnly bool           `gobom:"set to true to include only main packages in recursive mode"`
	GomodExcludes *regexp.Regexp `gobom:"regexp of paths to exclude"`
}

func init() {
	gobom.RegisterGenerator(&Generator{
		GomodTests:    true,
		GomodPackages: false,
	})
}

// Configure sets the options for this Generator
func (g *Generator) Configure() error {
	filters := make(map[string]bool)
	for _, name := range g.Filters {
		filters[name] = true
	}

	g.GomodTests = g.GomodTests && (!filters["release"] || filters["test"])

	if g.Excludes != nil {
		if g.GomodExcludes == nil {
			g.GomodExcludes = g.Excludes
		} else {
			g.GomodExcludes = regexp.MustCompile(g.GomodExcludes.String() + "|" + g.Excludes.String())
		}
	}

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

	if !g.GomodPackages {
		for _, module := range modules {
			module.Components = nil
		}
	}

	return &cyclonedx.BOM{Components: modules}, nil
}

func (g *Generator) listPackages(path string) ([]*cyclonedx.Component, error) {
	packages := make([]*cyclonedx.Component, 0)

	log.Info("listing package dependencies in '%s'", path)

	args := []string{"list", "-mod", "readonly", "-deps", "-json"}
	if g.GomodTests {
		args = append(args, "-test")
	}
	paths, err := g.listPackagePaths(path)
	if err != nil {
		return nil, fmt.Errorf("go list: %v", err)
	}
	args = append(args, paths...)

	cmd := exec.Command("go", args...)
	cmd.Dir = path
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("go list: %v", err)
	}

	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("go list start: %v", err)
	}

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
		log.Error("'go list' failed with error message:\n\n%s", stderr.String())
		return nil, fmt.Errorf("go list: %v", err)
	}

	return packages, nil
}

func (g *Generator) listPackagePaths(path string) ([]string, error) {
	if !g.Recurse {
		return []string{"."}, nil
	}

	if !g.GomodMainOnly && g.GomodExcludes == nil {
		return []string{"./..."}, nil
	}

	log.Debug("listing packages")

	args := []string{"list", "-mod", "readonly", "-f"}
	if g.GomodMainOnly {
		args = append(args, `{{if eq .Name "main"}}{{.Dir}}{{end}}`, "./...")
	} else {
		args = append(args, "{{.Dir}}", "./...")
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	paths := strings.Split(strings.Trim(string(out), "\n"), "\n")
	matches := make([]string, 0, len(paths))
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	for _, p := range paths {
		match, err := filepath.Rel(path, p)
		if err != nil {
			return nil, err
		}

		if g.GomodExcludes == nil || !g.GomodExcludes.MatchString(match) {
			matches = append(matches, "."+string(filepath.Separator)+match)
		} else {
			log.Debug("skipping '%s'", match)
		}
	}

	log.Debug("found %d matching package(s): %s", len(matches), strings.Join(matches, ", "))
	return matches, nil
}

func resolveGoVersion(modules []*cyclonedx.Component) error {
	log.Debug("resolving Go version")

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
	log.Trace("local Go version is %s", version)
	for _, module := range modules {
		if module.Name == "github.com/golang/go" && module.Version == "unknown" {
			module.Version = version
			module.PURL = gobom.PURL(gobom.GolangPackage, module.Name, version)
		}
	}

	return nil
}

func resolveWhy(path string, modules []*cyclonedx.Component) error {
	log.Debug("resolving 'why' for all modules")

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
				Description: "Golang module\n\nPackages:\n",
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
		return "unknown"
	} else if strings.HasPrefix(version, "v") {
		return version[1:]
	}

	return version
}
