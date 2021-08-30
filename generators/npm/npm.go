package npm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/cyclonedx"
	"github.com/mattermost/gobom/log"
)

// Generator generates BOMs for npm projects
type Generator struct {
	gobom.BaseGenerator

	NpmDevDependencies bool           `gobom:"set to true to include devDependencies"`
	NpmExcludes        *regexp.Regexp `gobom:"regexp of paths to exclude"`
}

func init() {
	gobom.RegisterGenerator(&Generator{})
}

// Configure sets the options for this Generator
func (g *Generator) Configure() error {
	filters := make(map[string]bool)
	for _, name := range g.Filters {
		filters[name] = true
	}

	g.NpmDevDependencies = g.NpmDevDependencies && (!filters["release"] || filters["test"])

	if g.Excludes != nil {
		if g.NpmExcludes == nil {
			g.NpmExcludes = g.Excludes
		} else {
			g.NpmExcludes = regexp.MustCompile(g.NpmExcludes.String() + "|" + g.Excludes.String())
		}
	}

	return nil
}

// GenerateBOM returns a CycloneDX BOM for the specified package path
func (g *Generator) GenerateBOM(path string) (*cyclonedx.BOM, error) {
	bom := &cyclonedx.BOM{}

	if g.Recurse {
		lockfiles, err := g.readLockfiles(path)
		if err != nil {
			return nil, err
		}
		for _, lockfile := range lockfiles {
			tree := g.generateComponentTree(lockfile)
			bom.Components = append(bom.Components, flatten(tree)...)
		}
	} else {
		lockfile, err := readLockfile(path)
		if err != nil {
			return nil, err
		}
		tree := g.generateComponentTree(lockfile)
		bom.Components = flatten(tree)
	}

	return bom, nil
}

func flatten(tree *npmComponent) []*cyclonedx.Component {
	describe(tree)
	components := []*cyclonedx.Component{&tree.Component}
	for _, subtree := range tree.installed {
		components = append(components, flatten(subtree)...)
	}
	return components
}

func describe(component *npmComponent) {
	component.Description = "npm package\n"

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

func buildDependencyChains(component *npmComponent, maxDepth int) [][]string {
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

func (g *Generator) generateComponentTree(lockfile *lockfile) *npmComponent {
	requires := make(map[string]string)
	for key, value := range lockfile.manifest.DevDependencies {
		requires[key] = value
	}
	for key, value := range lockfile.manifest.Dependencies {
		requires[key] = value
	}
	return g.generateComponentSubtree(lockfile.Name, &dependency{
		Version:      lockfile.Version,
		Requires:     requires,
		Dependencies: lockfile.Dependencies,
	}, nil)
}

func (g *Generator) generateComponentSubtree(name string, pkg *dependency, parent *npmComponent) *npmComponent {
	// The npm lockfile holds an exact representation of the directory structure
	// any dependencies will be installed in. It does not, however, directly show
	// why a specific dependency was included in the first place: a dependency could
	// be installed in one of the parent directories of the package that actually
	// required it. The dependency graph is still something we're interested in.
	//
	// To resolve the dependency graph, we first build the CycloneDX Components for
	// all dependencies in the order they are installed. Then we walk the components
	// and resolve their relationships by following the node import path search order
	//
	component := &npmComponent{}
	component.Type = cyclonedx.Library
	component.Name = path.Base(name)
	component.Group = path.Dir(name)
	if component.Group == "." {
		component.Group = ""
	}
	component.Version = pkg.Version
	component.PURL = gobom.PURL(gobom.NpmPackage, name, component.Version)
	component.parent = parent
	component.requires = pkg.Requires
	if parent == nil {
		component.root = true
	}

	// generate npmComponents for all installed dependencies
	component.installed = make(map[string]*npmComponent)
	for name, info := range pkg.Dependencies {
		if g.NpmDevDependencies || !info.Dev {
			component.installed[name] = g.generateComponentSubtree(name, info, component)
		}
	}

	// resolve the dependency hierarchy
	resolveDependants(component)

	return component
}

func resolveDependants(component *npmComponent) {
	// follow the node import path search order to resolve dependants
	for name := range component.requires {
		for ancestor := component; ancestor != nil; ancestor = ancestor.parent {
			if dependency, exists := ancestor.installed[name]; exists {
				dependency.dependants = append(dependency.dependants, component)
				break
			}
		}
	}

	// resolve recursively
	for _, dependency := range component.installed {
		resolveDependants(dependency)
	}
}

func readLockfile(path string) (*lockfile, error) {
	data, err := ioutil.ReadFile(filepath.Join(path, "package-lock.json"))
	if err != nil {
		data, err = ioutil.ReadFile(filepath.Join(path, "npm-shrinkwrap.json"))
		if err != nil {
			return nil, err
		}
		log.Info("read 'npm-shrinkwrap.json' in '%s'", path)
	} else {
		log.Info("read 'package-lock.json' in '%s'", path)
	}
	lockfile := &lockfile{}
	err = json.Unmarshal(data, lockfile)
	if err != nil {
		return nil, err
	}
	if lockfile.Name == "" {
		lockfile.Name = path
	}
	if lockfile.Version == "" {
		lockfile.Version = "unknown"
	}

	// read package.json if available
	data, err = ioutil.ReadFile(filepath.Join(path, "package.json"))
	if err == nil {
		log.Trace("read 'package.json' in '%s'", path)
		errUnMarshal := json.Unmarshal(data, &lockfile.manifest)
		if errUnMarshal != nil {
			return lockfile, errUnMarshal
		}
	}

	return lockfile, err
}

func (g *Generator) readLockfiles(path string) (map[string]*lockfile, error) {
	lockfiles := make(map[string]*lockfile)

	if g.NpmExcludes != nil && g.NpmExcludes.MatchString(path) {
		log.Debug("skipping '%s'", path)
		return lockfiles, nil
	}

	// traverse subdirectories
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		if info.IsDir() {
			if info.Name() == "node_modules" {
				log.Debug("skipping 'node_modules' directory in '%s'", path)
				continue
			}
			lockfiles2, err := g.readLockfiles(filepath.Join(path, info.Name()))
			if err != nil {
				return nil, err
			}
			for key, value := range lockfiles2 {
				lockfiles[key] = value
			}
		}
	}

	// get the lockfile for this directory
	lockfile, err := readLockfile(path)
	if err == nil {
		lockfiles[path] = lockfile
	}
	return lockfiles, nil
}

type lockfile struct {
	Name         string
	Version      string
	Dependencies map[string]*dependency

	manifest struct {
		Dependencies    map[string]string
		DevDependencies map[string]string
	}
}

type dependency struct {
	Version      string
	Dev          bool
	Optional     bool
	Requires     map[string]string
	Dependencies map[string]*dependency
}

type npmComponent struct {
	cyclonedx.Component

	parent     *npmComponent            // the component under which this component was installed
	installed  map[string]*npmComponent // components installed under this component
	requires   map[string]string        // package names and semver ranges this component depends on
	dependants []*npmComponent          // components dependant on this component
	root       bool                     // is this the root of a component tree?
}
