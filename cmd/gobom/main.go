package main

import (
	"github.com/mattermost/gobom/commands"

	_ "github.com/mattermost/gobom/generators"
)

func main() {
	_ = commands.Execute()
}
