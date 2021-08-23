package generators

import (
	// allow importing all default generators with a single import
	_ "github.com/mattermost/gobom/generators/cocoapods"
	_ "github.com/mattermost/gobom/generators/gclient"
	_ "github.com/mattermost/gobom/generators/gomod"
	_ "github.com/mattermost/gobom/generators/gradle"
	_ "github.com/mattermost/gobom/generators/npm"
)
