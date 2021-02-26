package commands

import (
	"github.com/mattermost/gobom/commands/internal/generate"
	"github.com/mattermost/gobom/commands/internal/upload"
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

var rootCmd = &cobra.Command{
	Use:   "gobom [command]",
	Short: "generate software bills of materials for various programming languages and ecosystems",
}

func init() {
	rootCmd.PersistentFlags().CountVarP((*int)(&log.LogLevel), "verbose", "v", "enable verbose logging")
	rootCmd.AddCommand(generate.Command)
	rootCmd.AddCommand(upload.Command)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
