package prerun

import (
	"encoding/json"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/mattermost/gobom/log"
	"github.com/spf13/cobra"
)

// Configure reads and applies the config file if specified
func Configure(config string, cmd *cobra.Command) bool {
	file, err := ioutil.ReadFile(config)
	if err != nil {
		log.Error("unable to read config file: %v", err)
		return false
	}
	flags := make(map[string]interface{})
	err = json.Unmarshal(file, &flags)
	if err != nil {
		log.Error("unable to parse config file: %v", err)
		return false
	}
	for name, value := range flags {
		flag := cmd.Flag(name)
		if flag == nil {
			log.Error("unrecognized flag name in config file: '%s'", name)
			return false
		}
		if flag.Changed {
			log.Debug("config '%s' overridden from command line", name)
		} else {
			flagValue := toFlagValue(value)
			log.Trace("setting config '%s' to '%s'", name, flagValue)
			flag.Value.Set(flagValue)
		}
	}

	return true
}

func toFlagValue(value interface{}) string {
	switch t := value.(type) {
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		// treating blindly as int since floats aren't used anywhere
		return strconv.Itoa(int(t))
	case string:
		return t
	case []interface{}:
		values := make([]string, 0, len(t))
		for _, value := range t {
			values = append(values, toFlagValue(value))
		}
		return strings.Join(values, ",")
	case map[string]interface{}:
		values := make([]string, 0, len(t))
		for key, value := range t {
			values = append(values, key+"="+toFlagValue(value))
		}
		return strings.Join(values, ",")
	default:
		panic("unexpected flag value type")
	}
}
