package main

import (
	"github.com/mattermost/gobom/commands"

	_ "github.com/mattermost/gobom/examples/custom_generator/helloworld" // our custom implementation
	_ "github.com/mattermost/gobom/generators"                           // default generators
)

func main() {
	commands.Execute()
}
