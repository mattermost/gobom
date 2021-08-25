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
	"github.com/spf13/pflag"
)

var (
	generators       []string
	properties       []string
	globalProperties = make(map[string]string)
)

// Command .
var Command = &cobra.Command{
	Use:   "generate [flags] [path]",
	Short: "generate software bills of materials",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.LogLevel += log.LevelWarn
		path := args[0] // TODO: default to ".", allow specifying multiple paths

		properties := sliceToMap(properties)
		for name, propname := range globalProperties {
			flag := cmd.Flags().Lookup(name)
			if !flag.Changed {
				continue
			}
			log.Trace("setting property '%s' from flag '%s'", propname, name)
			if slice, ok := flag.Value.(pflag.SliceValue); ok {
				properties[propname] = strings.Join(slice.GetSlice(), ",")
			} else {
				properties[propname] = flag.Value.String()
			}
		}
		configuredGenerators := make(map[string]gobom.Generator)
		if len(generators) == 0 {
			// default to running all generators
			log.Debug("configuring available generators")
			availableGenerators := gobom.Generators()
			for name, generator := range availableGenerators {
				if err := configure(generator, properties); err != nil {
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
					if err := configure(generator, properties); err != nil {
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

		log.Debug("merging and marshaling BOMs")

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
	Command.Flags().StringSliceVarP(&generators, "generators", "g", []string{}, "commma-separated list of generators to run")
	Command.Flags().StringSliceVarP(&properties, "properties", "p", []string{}, "properties to pass to generators in the form 'Prop1Name=val1,Prop2Name=val2")

	// inherit flags from the upload command
	Command.Flags().AddFlagSet(upload.Command.Flags())

	// register flags from generators
	gobom.OnGeneratorRegistered(registerGeneratorFlags)
}

func registerGeneratorFlags(_ string, g gobom.Generator) {
	name := gobom.ResolveShortName(g)
	gobom.VisitProperties(g, func(field reflect.StructField, value reflect.Value) {
		if !strings.HasPrefix(strings.ToLower(field.Name), name) {
			parts := strings.SplitN(field.Tag.Get("gobom"), ",", 3)
			if len(parts) != 3 {
				panic(fmt.Sprintf("bad gobom tag on unprefixed field in '%s'", gobom.ResolveName(g)))
			}
			if parts[0] != "" {
				if flag := Command.Flags().Lookup(parts[0]); flag != nil {
					if flag.Name == parts[0] && flag.Shorthand == parts[1] && flag.Usage == parts[2] {
						// flag already registered and tag matches; move on
						return
					}
					panic(fmt.Sprintf("conflicting gobom tag on unprefixed field in '%s'", gobom.ResolveName(g)))
				}
				switch field.Type {
				case reflect.TypeOf(true):
					Command.Flags().BoolP(parts[0], parts[1], false, parts[2])
				case reflect.TypeOf(""):
					Command.Flags().StringP(parts[0], parts[1], "", parts[2])
				case reflect.TypeOf([]string{}):
					Command.Flags().StringSliceP(parts[0], parts[1], []string{}, parts[2])
				case reflect.TypeOf(regexp.MustCompile("")):
					Command.Flags().StringP(parts[0], parts[1], "", parts[2])
				default:
					panic(fmt.Sprintf("unsupported property type %s", field.Type))
				}
				globalProperties[parts[0]] = field.Name
			}
		}
	})
}

func configure(generator gobom.Generator, properties map[string]string) error {
	var errs []error

	gobom.VisitProperties(generator, func(field reflect.StructField, value reflect.Value) {
		if prop, exists := properties[field.Name]; exists && field.Tag.Get("gobom") != "" {
			switch field.Type {
			case reflect.TypeOf(true):
				switch prop {
				case "true":
					value.Set(reflect.ValueOf(true))
				case "false":
					value.Set(reflect.ValueOf(false))
				default:
					errs = append(errs, fmt.Errorf("unsupported boolean value '%s'", prop))
				}
			case reflect.TypeOf(""):
				value.Set(reflect.ValueOf(prop))
			case reflect.TypeOf([]string{}):
				value.Set(reflect.ValueOf(strings.Split(prop, ":")))
			case reflect.TypeOf(regexp.MustCompile("")):
				pattern, err := regexp.Compile(prop)
				if err != nil {
					errs = append(errs, err)
				}
				value.Set(reflect.ValueOf(pattern))
			default:
				panic(fmt.Sprintf("unsupported property type %s", field.Type))
			}
		}
	})

	if len(errs) != 0 {
		return errs[0]
	}

	return generator.Configure()
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
