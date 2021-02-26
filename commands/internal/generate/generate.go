package generate

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/cyclonedx"
	"github.com/mattermost/gobom/log"
	"github.com/spf13/cobra"
)

var (
	subcomponents bool
	tests         bool
	recurse       bool
	generators    []string
	properties    []string
)

// Command .
var Command = &cobra.Command{
	Use:   "generate [flags] [path]",
	Short: "generate software bills of materials for various programming languages and ecosystems",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.LogLevel += log.LevelWarn

		path := args[0]
		options := gobom.Options{
			IncludeSubcomponents: subcomponents,
			IncludeTests:         tests,
			Recurse:              recurse,
			Properties:           sliceToMap(properties),
		}
		availableGenerators := gobom.Generators()
		configuredGenerators := make(map[string]gobom.Generator)
		if len(generators) == 0 {
			// default to running all generators
			log.Debug("configuring available generators")
			for name, generator := range availableGenerators {
				if err := generator.Configure(options); err != nil {
					log.Warn("configuring '%s' generator  failed: %v", name, err)
				} else {
					configuredGenerators[name] = generator
				}
			}
		} else {
			// run only specific generators
			log.Debug("configuring generators: %s", strings.Join(generators, ", "))
			for _, name := range generators {
				generator, ok := availableGenerators[name]
				if ok {
					if err := generator.Configure(options); err != nil {
						log.Warn("configuring '%s' generator failed: %v", name, err)
					} else {
						configuredGenerators[name] = generator
					}
				} else {
					log.Warn("no such generator: %s", name)
				}
			}
		}

		boms := make([]*cyclonedx.BOM, 0, len(configuredGenerators))
		for name, generator := range configuredGenerators {
			log.Info("running '%s' generator", name)
			bom, err := generator.GenerateBOM(path)
			if err != nil {
				log.Warn("'%s' generator returned an error: %v", name, err)
			}
			if bom != nil {
				boms = append(boms, bom)
			}
		}

		log.Debug("meging and marshaling BOMs")

		out, _ := xml.Marshal(merge(boms))
		fmt.Println(xml.Header + string(out))
	},
}

func init() {
	Command.Flags().BoolVarP(&subcomponents, "subcomponents", "s", false, "include subcomponents in the output")
	Command.Flags().BoolVarP(&tests, "tests", "t", false, "include dependencies only required for testing/development")
	Command.Flags().BoolVarP(&recurse, "recurse", "r", false, "scan the target path recursively")
	Command.Flags().StringSliceVarP(&generators, "generators", "g", []string{}, "commma-separated list of generators to run")
	Command.Flags().StringSliceVarP(&properties, "properties", "p", []string{}, "properties to pass to generators in the form 'Prop1Name=val1,Prop2Name=val2")
}

func merge(parts []*cyclonedx.BOM) *cyclonedx.BOM {
	count := 0
	for _, part := range parts {
		count += len(part.Components)
	}
	bom := &cyclonedx.BOM{
		Components: make([]*cyclonedx.Component, 0, count),
	}
	for _, part := range parts {
		bom.Components = append(bom.Components, part.Components...)
	}
	return bom
}

func sliceToMap(slice []string) map[string]string {
	m := make(map[string]string)
	for _, value := range slice {
		i := strings.IndexRune(value, '=')
		if i == -1 {
			m[value] = ""
		} else {
			m[value[:i]] = value[i+1:]
		}
	}
	return m
}
