package generate

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/commands/internal/upload"
	"github.com/mattermost/gobom/cyclonedx"
	"github.com/mattermost/gobom/log"
	"github.com/spf13/cobra"
)

var (
	recurse    bool
	generators []string
	properties []string
)

// Command .
var Command = &cobra.Command{
	Use:   "generate [flags] [path]",
	Short: "generate software bills of materials",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.LogLevel += log.LevelWarn

		path := args[0] // TODO: default to ".", allow specifying multiple paths
		options := gobom.Options{
			Recurse: recurse,
		}
		properties := sliceToMap(properties)
		configuredGenerators := make(map[string]gobom.Generator)
		if len(generators) == 0 {
			// default to running all generators
			log.Debug("configuring available generators")
			availableGenerators := gobom.Generators()
			for name, generator := range availableGenerators {
				if err := configure(generator, options, properties); err != nil {
					log.Warn("configuring '%s'  failed: %v", name, err)
				} else {
					configuredGenerators[name] = generator
				}
			}
		} else {
			// run only specific generators
			log.Debug("configuring generators: %s", strings.Join(generators, ", "))
			for _, name := range generators {
				generator, err := gobom.GetGenerator(name)
				if err == nil {
					if err := configure(generator, options, properties); err != nil {
						log.Warn("configuring '%s' generator failed: %v", name, err)
					} else {
						configuredGenerators[name] = generator
					}
				} else {
					log.Warn(err.Error())
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

		if cmd.Flag("url").Value.String() != "" {
			// suppress output and upload directly to Dependency-Track
			buffer := &bytes.Buffer{}
			buffer.WriteString(xml.Header)
			encoder := xml.NewEncoder(buffer)
			encoder.Encode(merge(boms))
			upload.Upload(buffer)
		} else {
			// no upload, just print to stdout
			out, _ := xml.Marshal(merge(boms))
			fmt.Println(xml.Header + string(out))
		}
	},
}

func init() {
	Command.Flags().BoolVarP(&recurse, "recurse", "r", false, "scan the target path recursively")
	Command.Flags().StringSliceVarP(&generators, "generators", "g", []string{}, "commma-separated list of generators to run")
	Command.Flags().StringSliceVarP(&properties, "properties", "p", []string{}, "properties to pass to generators in the form 'Prop1Name=val1,Prop2Name=val2")

	// inherit flags from the upload command
	Command.Flags().AddFlagSet(upload.Command.Flags())
}

func configure(generator gobom.Generator, options gobom.Options, properties map[string]string) error {
	g := reflect.ValueOf(generator).Elem()
	t := g.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if prop, exists := properties[field.Name]; exists && field.Tag.Get("gobom") != "" {
			switch field.Type {
			case reflect.TypeOf(true):
				switch prop {
				case "true":
					g.Field(i).Set(reflect.ValueOf(true))
				case "false":
					g.Field(i).Set(reflect.ValueOf(false))
				default:
					return fmt.Errorf("unsupported boolean value '%s'", prop)
				}
			case reflect.TypeOf(""):
				g.Field(i).Set(reflect.ValueOf(prop))
			case reflect.TypeOf([]string{}):
				g.Field(i).Set(reflect.ValueOf(strings.Split(prop, ":")))
			case reflect.TypeOf(regexp.MustCompile("")):
				pattern, err := regexp.Compile(prop)
				if err != nil {
					return err
				}
				g.Field(i).Set(reflect.ValueOf(pattern))
			default:
				panic(fmt.Sprintf("unsupported property type %s", field.Type))
			}
		}
	}
	return generator.Configure(options)
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
