package main

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/cocoapods"
	"github.com/mattermost/gobom/cyclonedx"
	"github.com/mattermost/gobom/gomod"
	"github.com/mattermost/gobom/gradle"
	"github.com/mattermost/gobom/log"
	"github.com/mattermost/gobom/npm"
	"github.com/spf13/cobra"
)

func main() {

	var (
		subcomponents bool
		tests         bool
		recurse       bool
		generators    []string
		properties    []string
	)

	cmd := &cobra.Command{
		Use:   "gobom [flags] [path]",
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
			availableGenerators := map[string]gobom.Generator{
				"go":        &gomod.Generator{},
				"npm":       &npm.Generator{},
				"cocoapods": &cocoapods.Generator{},
				"gradle":    &gradle.Generator{},
			}
			configuredGenerators := make([]gobom.Generator, 0, len(availableGenerators))
			if len(generators) == 0 {
				// default to running all generators
				log.Debug("configuring available generators")
				for _, generator := range availableGenerators {
					if err := generator.Configure(options); err != nil {
						log.Warn("configuring %s failed: %v", generator.Name(), err)
					} else {
						configuredGenerators = append(configuredGenerators, generator)
					}
				}
			} else {
				// run only specific generators
				log.Debug("configuring generators: %s", strings.Join(generators, ", "))
				for _, name := range generators {
					generator, ok := availableGenerators[name]
					if ok {
						if err := generator.Configure(options); err != nil {
							log.Warn("configuring %s failed: %v", generator.Name(), err)
						} else {
							configuredGenerators = append(configuredGenerators, generator)
						}
					} else {
						log.Warn("no such generator: %s", name)
					}
				}
			}

			boms := make([]*cyclonedx.BOM, 0, len(configuredGenerators))
			for _, generator := range configuredGenerators {
				log.Debug("running %s", generator.Name())
				bom, err := generator.GenerateBOM(path)
				if err != nil {
					log.Warn("%s returned an error: %v", generator.Name(), err)
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
	cmd.Flags().BoolVarP(&subcomponents, "subcomponents", "s", false, "include subcomponents in the output")
	cmd.Flags().BoolVarP(&tests, "tests", "t", false, "include dependencies only required for testing/development")
	cmd.Flags().BoolVarP(&recurse, "recurse", "r", false, "scan the target path recursively")
	cmd.Flags().StringSliceVarP(&generators, "generators", "g", []string{}, "commma-separated list of generators to run")
	cmd.Flags().StringSliceVarP(&properties, "properties", "p", []string{}, "properties to pass to generators in the form 'Prop1Name=val1,Prop2Name=val2")
	cmd.Flags().CountVarP((*int)(&log.LogLevel), "verbose", "v", "enable verbose logging")
	cmd.Execute()
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
