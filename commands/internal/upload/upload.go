package upload

import (
	"github.com/spf13/cobra"
)

// Command .
var Command = &cobra.Command{
	Use:   "upload [flags]",
	Short: "upload a BOM file to Dependency-Track",
	Run:   func(cmd *cobra.Command, args []string) {},
}
