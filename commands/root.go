package commands

import (
	"github.com/mattermost/gobom/commands/internal/generate"
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
	rootCmd.AddCommand(generate.Command)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
