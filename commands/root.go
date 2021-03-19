package commands

import (
	"fmt"
	"os"
	"reflect"

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

func buildGeneratorHelpText(key string, g gobom.Generator) string {
	props := make(map[string]string)
	t := reflect.ValueOf(g).Elem().Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		help := field.Tag.Get("gobom")
		if help != "" {
			props[field.Name] = help
		}
	}
	propHelp := ""
	if len(props) > 0 {
		propHelp = "Available Properties:\n"
		for name, help := range props {
			propHelp = fmt.Sprintf("%s  %-24s %s\n", propHelp, name, help)
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
