package gradle

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/cyclonedx"
	"github.com/mattermost/gobom/log"
)

// Generator generates BOMs for CocoaPods projects
type Generator struct {
	options gobom.Options
	paths   []string
	exclude *regexp.Regexp
}

func init() {
	gobom.RegisterGenerator(&Generator{})
}

// Configure sets the options for this Generator
func (g *Generator) Configure(options gobom.Options) error {
	g.options = options

	if paths, ok := options.Properties["GradlePath"]; ok {
		g.paths = strings.Split(paths, ":")
	} else {
		// default to not using the wrapper; safe on untrusted code
		g.paths = []string{"gradle"}
	}

	if exclude, ok := options.Properties["GradleExcludes"]; ok {
		var err error
		g.exclude, err = regexp.Compile(exclude)
		if err != nil {
			log.Error("invalid exclude pattern: '%s'", exclude)
		}
	}

	return nil
}

// GenerateBOM returns a CycloneDX BOM for the specified package path
func (g *Generator) GenerateBOM(path string) (*cyclonedx.BOM, error) {

	var err error
	bom := &cyclonedx.BOM{}
	if g.options.Recurse {
		bom.Components, err = g.generateComponentsRecursively(path)
		if err != nil {
			return nil, err
		}
	} else {
		bom.Components, err = g.generateComponents(path)
		if err != nil {
			return nil, err
		}
	}
	return bom, nil
}

func (g *Generator) generateComponentsRecursively(path string) ([]*cyclonedx.Component, error) {
	components, _ := g.generateComponents(path)

	// traverse subdirectories
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		if info.IsDir() {
			next := filepath.Join(path, info.Name())
			if g.exclude != nil && g.exclude.MatchString(next) {
				log.Debug("skipping '%s'", next)
				continue
			}
			components2, err := g.generateComponentsRecursively(next)
			if err == nil {
				components = append(components, components2...)
			}
		}
	}
	return components, nil
}

func (g *Generator) generateComponents(path string) ([]*cyclonedx.Component, error) {
	_, err := os.Stat(filepath.Join(path, "build.gradle"))
	if err != nil {
		return nil, err
	}

	configs, err := g.listDependencies(path)
	if err != nil {
		log.Warn("listing depdendencies failed: %v", err)
		return nil, err
	}

	log.Debug("parsing dependency hierarchy")
	components := map[string]*gradleComponent{}
	for _, config := range configs {
		config.Walk(func(dependency, dependant *dependency) {
			parsed := dependency.Parse()
			if parsed.resolved {
				component, exists := components[parsed.PURL]
				if !exists {
					component = parsed
					components[component.PURL] = component
				}
				if component.configs == nil {
					component.configs = make(map[string]bool)
				}
				component.configs[config.Name()] = true
				if dependant != nil {
					component.dependants = append(component.dependants, components[dependant.Parse().PURL])
				} else {
					// this is a direct dependency of the build configuration
					// include the build config itself as a component to make it visible
					configComponent, ok := components[config.Name()]
					if !ok {
						configComponent = &gradleComponent{}
						configComponent.Type = cyclonedx.Library
						configComponent.Name = config.Name()
						components[configComponent.Name] = configComponent
					}
					component.dependants = append(component.dependants, configComponent)
				}
			}
		})
	}

	result := make([]*cyclonedx.Component, 0, len(components))
	for _, component := range components {
		describe(component)
		result = append(result, &component.Component)
	}

	return result, nil
}

func describe(component *gradleComponent) {
	if component.PURL == "" {
		log.Trace("building description for '%s'", component.Name)
	} else {
		log.Trace("building description for '%s'", component.PURL)
	}
	if len(component.dependants) == 0 {
		component.Description = "Gradle build configuration\n"
		return
	}
	if component.project {
		component.Description = "Gradle project\n"
	} else {
		component.Description = "Gradle dependency\n"
	}

	configs := make([]string, 0, len(component.configs))
	for config := range component.configs {
		configs = append(configs, config)
	}
	component.Description = fmt.Sprintf("%s\nAppears in: %s\n", component.Description, strings.Join(configs, ", "))

	min := 0
	shortestChain := []string{}
	for _, chain := range buildDependencyChains(component, 3) {
		if min == 0 || len(chain) < min {
			min = len(chain)
			shortestChain = chain
		}
	}
	component.Description = fmt.Sprintf("%s\nRequired by:\n\t%s\n", component.Description, strings.Join(shortestChain, "\n\t"))

}

func buildDependencyChains(component *gradleComponent, maxDepth int) [][]string {
	if len(component.dependants) == 0 || maxDepth < 0 {
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

func (g *Generator) gradle(wd string) (string, error) {
	for _, path := range g.paths {
		if !filepath.IsAbs(path) && path != filepath.Base(path) {
			wd, err := filepath.Abs(wd)
			if err != nil {
				return "", err
			}
			path = filepath.Join(wd, path)
		}
		path, err := exec.LookPath(path)
		if err == nil {
			log.Debug("using Gradle binary from '%s'", path)
			return path, nil
		}
	}
	return "", fmt.Errorf("could not locate Gradle binary")
}

func (g *Generator) listDependencies(path string) ([]*buildConfig, error) {
	log.Info("listing dependencies in '%s'", path)

	gradle, err := g.gradle(path)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(gradle, "-q", "--console", "plain", "dependencies")
	cmd.Dir = path
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	out := unmarshal(stdout)
	trees := []*buildConfig{}
	for _, node := range out {
		if len(node.nodes) > 0 {
			trees = append(trees, (*buildConfig)(node))
		}
	}
	err = cmd.Wait()
	return trees, err
}

type node struct {
	value string
	nodes []*node
}

func unmarshal(r io.Reader) []*node {
	output := []*node{}
	reader := bufio.NewReader(r)

	var (
		node *node
		err  error
	)
	for err == nil {
		node, err = unmarshalSubtree(reader, 0)
		output = append(output, node)
	}
	return output
}

var nodePrefix = regexp.MustCompile(`^([| ]    )*[\\+]--- $`)

func unmarshalSubtree(reader *bufio.Reader, depth int) (*node, error) {
	var err error
	tree := &node{}
	if depth > 0 {
		peek, err := reader.Peek(depth * 5)
		if err != nil {
			return nil, err
		}
		if nodePrefix.Match(peek) {
			reader.Discard(len(peek))
		} else {
			return nil, nil
		}
	}
	tree.value, err = reader.ReadString('\n')
	if err != nil {
		return tree, err
	}
	for {
		subtree, err := unmarshalSubtree(reader, depth+1)
		if subtree == nil {
			return tree, err
		}
		tree.nodes = append(tree.nodes, subtree)
		if err != nil {
			return tree, err
		}
	}
}

type buildConfig node

var buildConfigValue = regexp.MustCompile(`^(.*?) - (.*)\n`)

func (t *buildConfig) Name() string {
	return buildConfigValue.FindStringSubmatch(t.value)[1]
}

func (t *buildConfig) Description() string {
	return buildConfigValue.FindStringSubmatch(t.value)[2]
}

func (t *buildConfig) Walk(cb func(*dependency, *dependency)) {
	for _, node := range t.nodes {
		dependency := (*dependency)(node)
		cb(dependency, nil)
		dependency.Walk(cb)
	}
}

type dependency node

func (d *dependency) Parse() *gradleComponent {
	result := &gradleComponent{}
	result.Type = cyclonedx.Library
	value := d.value
parseDependency:
	result.resolved = true
	if strings.HasPrefix(value, "project ") {
		result.project = true
		value = value[8:]
	}
	if i := strings.IndexRune(value, ':'); i != -1 {
		result.Group = value[:i]
		value = value[i+1:]
	}
	if i := strings.IndexAny(value, ": \n"); i != -1 {
		result.Name = value[:i]
		value = value[i+1:]
	} else {
		log.Error("unable to parse dependency value: '%s'", d.value)
		result.Name = "unknown"
		result.Version = "unknown"
		return result
	}
	if i := strings.Index(value, " -> "); i != -1 {
		value = value[i+4:]
		if strings.Contains(value, ":") {
			goto parseDependency
		}
	}
	if strings.HasSuffix(value, "(*)\n") || strings.HasSuffix(value, "(c)\n") {
		value = value[:len(value)-4]
	} else if strings.HasSuffix(value, "(n)\n") {
		result.resolved = false
		value = value[:len(value)-4]
	}
	result.Version = strings.TrimSpace(value)
	if result.Version == "" {
		result.Version = "unknown"
	}

	if result.Group != "" {
		result.PURL = gobom.PURL(gobom.GradlePackage,
			fmt.Sprintf("%s/%s", result.Group, result.Name),
			result.Version)
	} else {
		result.PURL = gobom.PURL(gobom.GradlePackage, result.Name, result.Version)
	}

	return result
}

func (d *dependency) Walk(cb func(*dependency, *dependency)) {
	for _, node := range d.nodes {
		dependency := (*dependency)(node)
		cb(dependency, d)
		dependency.Walk(cb)
	}
}

type gradleComponent struct {
	cyclonedx.Component

	resolved   bool
	project    bool
	dependants []*gradleComponent
	configs    map[string]bool
}
