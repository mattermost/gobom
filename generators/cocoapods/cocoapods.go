package cocoapods

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/cyclonedx"
	"github.com/mattermost/gobom/log"
	"gopkg.in/yaml.v3"
)

// Generator generates BOMs for CocoaPods projects
type Generator struct {
	gobom.BaseGenerator

	CocoapodsExcludes *regexp.Regexp `gobom:"regexp of paths to exclude"`
}

func init() {
	gobom.RegisterGenerator(&Generator{})
}

// Configure sets the options for this Generator
func (g *Generator) Configure() error {
	if g.Excludes != nil {
		if g.CocoapodsExcludes == nil {
			g.CocoapodsExcludes = g.Excludes
		} else {
			g.CocoapodsExcludes = regexp.MustCompile(g.CocoapodsExcludes.String() + "|" + g.Excludes.String())
		}
	}

	return nil
}

// GenerateBOM returns a CycloneDX BOM for the specified package path
func (g *Generator) GenerateBOM(path string) (*cyclonedx.BOM, error) {
	var err error
	bom := &cyclonedx.BOM{}
	if g.Recurse {
		bom.Components, err = g.generateComponentsRecursively(path)
		if err != nil {
			return nil, err
		}
	} else {
		bom.Components, err = generateComponents(path)
		if err != nil {
			return nil, err
		}
	}
	return bom, nil
}

func (g *Generator) generateComponentsRecursively(path string) ([]*cyclonedx.Component, error) {
	if g.CocoapodsExcludes != nil && g.CocoapodsExcludes.MatchString(path) {
		log.Debug("skipping '%s'", path)
		return nil, nil
	}
	components, _ := generateComponents(path)

	// traverse subdirectories
	infos, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		if info.IsDir() {
			components2, err := g.generateComponentsRecursively(filepath.Join(path, info.Name()))
			if err == nil {
				components = append(components, components2...)
			}
		}
	}
	return components, nil
}

func generateComponents(path string) ([]*cyclonedx.Component, error) {
	lockfile, err := readLockfile(path)
	if err != nil {
		return nil, err
	}
	log.Info("read 'Podfile.lock' in '%s'", path)
	components := map[string]*podComponent{}
	root := &podComponent{}
	root.Type = cyclonedx.Library
	root.Name = path
	root.Version = "unknown"
	root.PURL = gobom.PURL(gobom.GenericPackage, path, "unknown")
	root.root = true
	for _, dependency := range lockfile.Dependencies {
		// extract "name" from "name (source)"
		name := strings.Fields(dependency)[0]
		root.requires = append(root.requires, name)
	}
	components[path] = root
	for i := range lockfile.Pods {
		pod := lockfile.Pods.Get(i)
		component := &podComponent{}
		component.Type = cyclonedx.Library
		component.Name = pod.Name()
		component.Version = pod.Version()
		component.PURL = gobom.PURL(gobom.CocoapodsPackage, component.Name, component.Version)
		component.requires = pod.Requires()
		components[component.Name] = component
	}
	for _, component := range components {
		for _, requires := range component.requires {
			dependency := components[requires]
			dependency.dependants = append(dependency.dependants, component)
		}
	}

	result := make([]*cyclonedx.Component, 0, len(components))
	for _, component := range components {
		describe(component)
		result = append(result, &component.Component)
	}
	return result, nil
}

func describe(component *podComponent) {
	if component.root {
		component.Description = "CocoaPods project root\n"
		return
	}
	component.Description = "CocoaPods package\n"

	min := 0
	shortestChain := []string{}
	for _, chain := range buildDependencyChains(component, 5) {
		if min == 0 || len(chain) < min {
			min = len(chain)
			shortestChain = chain
		}
	}
	component.Description = fmt.Sprintf("%s\nRequired by:\n\t%s", component.Description, strings.Join(shortestChain, "\n\t"))

}

func buildDependencyChains(component *podComponent, maxDepth int) [][]string {
	if component.root || maxDepth < 0 {
		return [][]string{[]string{}}
	}
	chains := [][]string{}
	for _, dependant := range component.dependants {
		chains2 := buildDependencyChains(dependant, maxDepth-1)
		for i := range chains2 {
			chains2[i] = append([]string{fmt.Sprintf("%s@%s", dependant.Name, dependant.Version)}, chains2[i]...)
		}
		chains = append(chains, chains2...)
	}
	return chains
}

func readLockfile(path string) (*lockfile, error) {
	data, err := os.ReadFile(filepath.Join(path, "Podfile.lock"))
	if err != nil {
		return nil, err
	}
	lockfile := &lockfile{}
	err = yaml.Unmarshal(data, lockfile)
	if err != nil {
		return nil, err
	}

	return lockfile, err
}

type podComponent struct {
	cyclonedx.Component
	root       bool
	requires   []string
	dependants []*podComponent
}

type lockfile struct {
	Pods         pods     `yaml:"PODS"`
	Dependencies []string `yaml:"DEPENDENCIES"`
}

type pods []interface{}
type pod interface {
	Name() string
	Version() string
	Requires() []string
}
type stringPod string
type mapPod map[string]interface{}

func (p pods) Get(i int) pod {
	pod := p[i]
	switch t := pod.(type) {
	case string:
		return stringPod(t)
	case map[string]interface{}:
		return mapPod(t)
	default:
		panic("bad lockfile")
	}
}

func (s stringPod) Name() string {
	// extract "name" from "name (version)"
	return strings.Fields(string(s))[0]
}

func (s stringPod) Version() string {
	// extract "version" from "name (version)"
	fields := strings.Fields(string(s))
	return fields[1][1 : len(fields[1])-1]
}

func (s stringPod) Requires() []string {
	return []string{}
}

func (m mapPod) Name() string {
	for s := range m {
		return stringPod(s).Name()
	}
	panic("bad lockfile")
}

func (m mapPod) Version() string {
	for s := range m {
		return stringPod(s).Version()
	}
	panic("bad lockfile")
}

func (m mapPod) Requires() []string {
	for _, values := range m {
		requires := make([]string, 0, len(values.([]interface{})))
		for _, s := range values.([]interface{}) {
			// extract "name" from "name (version)"
			requires = append(requires, strings.Fields(s.(string))[0])
		}
		return requires
	}

	panic("bad lockfile")
}
