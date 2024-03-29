package commands

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/mattermost/gobom"
	"github.com/mattermost/gobom/commands/internal/generate"
	"github.com/mattermost/gobom/commands/internal/prerun"
	"github.com/mattermost/gobom/commands/internal/upload"
	"github.com/mattermost/gobom/log"
	"github.com/spf13/cobra"
)

var config string

var rootCmd = &cobra.Command{
	Use:   "gobom [command]",
	Short: "generate software bills of materials",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if config != "" {
			if !prerun.Configure(config, cmd) {
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.PersistentFlags().CountVarP((*int)(&log.LogLevel), "verbose", "v", "enable verbose logging")
	rootCmd.PersistentFlags().StringVarP(&config, "config", "c", "", "read flags from a JSON config file")
	rootCmd.AddCommand(generate.Command)
	rootCmd.AddCommand(upload.Command)

	gobom.OnGeneratorRegistered(registerGeneratorHelpTopic)
}

func registerGeneratorHelpTopic(key string, g gobom.Generator) {
	var helpCommand *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == key {
			helpCommand = cmd
		}
	}
	if helpCommand == nil {
		helpCommand = &cobra.Command{}
		rootCmd.AddCommand(helpCommand)
	}
	helpCommand.Use = key
	helpCommand.Short = fmt.Sprintf("help for '%s'", key)
	helpCommand.Long = buildGeneratorHelpText(key, g)
}

var (
	propToFlag      = make(map[string]string)
	propToShorthand = make(map[string]string)
	propToHelptext  = make(map[string]string)
	needsReregister = make(map[string][]string)
)

func buildGeneratorHelpText(key string, g gobom.Generator) string {
	name := gobom.ResolveShortName(g)
	props := make(map[string]string)
	globalProps := make(map[string][]string)

	gobom.VisitProperties(g, func(field reflect.StructField, value reflect.Value) {
		if strings.HasPrefix(strings.ToLower(field.Name), name) {
			props[field.Name] = field.Tag.Get("gobom")
		} else {
			parts := strings.SplitN(field.Tag.Get("gobom"), ",", 3)
			if len(parts) != 3 {
				panic(fmt.Sprintf("bad gobom tag on unprefixed field in '%s'", gobom.ResolveName(g)))
			}
			globalProps[field.Name] = parts
		}
	})

	propHelp := ""
	propHelp = "Global Properties:\n"
	footnotes := []string{}
	for name, parts := range globalProps {
		if parts[0] == "" {
			parts[0] = propToFlag[name]
			parts[1] = propToShorthand[name]
			helptext := propToHelptext[name]
			if helptext != "" {
				footnotes = append(footnotes, parts[2])
				parts[2] = fmt.Sprintf("%s [%d]", helptext, len(footnotes))
			}
		}
		if parts[0] == "" {
			propHelp = fmt.Sprintf("%s  %-32s %s\n", propHelp, name, parts[2])
			needsReregister[name] = append(needsReregister[name], key)
			println("needs reregister: " + name)
		} else if parts[1] == "" {
			propHelp = fmt.Sprintf("%s  %-16s --%-13s %s\n", propHelp, name, parts[0], parts[2])
		} else {
			propHelp = fmt.Sprintf("%s  %-16s -%s, --%-9s %s\n", propHelp, name, parts[1], parts[0], parts[2])
		}
		if parts[0] != "" && propToFlag[name] == "" {
			propToFlag[name] = parts[0]
			propToShorthand[name] = parts[1]
			propToHelptext[name] = parts[2]
			for _, keyToReregister := range needsReregister[name] {
				generatorToReregister, _ := gobom.GetGenerator(keyToReregister)
				registerGeneratorHelpTopic(keyToReregister, generatorToReregister)
			}
		}
	}
	if len(footnotes) > 0 {
		propHelp = fmt.Sprintf("%s\n", propHelp)
		for i, note := range footnotes {
			propHelp = fmt.Sprintf("%s  [%d]: %s\n", propHelp, i+1, note)
		}
	}
	if len(props) > 0 {
		propHelp = fmt.Sprintf("%s\nLocal Properties:\n", propHelp)
		for name, help := range props {
			propHelp = fmt.Sprintf("%s  %-32s %s\n", propHelp, name, help)
		}
	}

	return fmt.Sprintf(`%s generator for use with the 'generate' command

Usage:
  gobom generate -g %s [flags] [-p properties] [path]

%s`, gobom.ResolveShortName(g), key, propHelp)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
