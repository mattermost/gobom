package prerun

import (
	"testing"

	"github.com/mattermost/gobom/commands/internal/generate"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	_ "github.com/mattermost/gobom/generators/gomod"
)

func TestConfigure(t *testing.T) {
	cmd := generate.Command

	if ok := Configure("./testdata/no-such-file.json", cmd); ok {
		t.Errorf("expected a failure on nonexistent config file")
	}
	reset(cmd)

	if ok := Configure("./testdata/bad-config.json", cmd); ok {
		t.Errorf("expected a failure on bad config file")
	}
	reset(cmd)

	if ok := Configure("./testdata/unparseable-config.json", cmd); ok {
		t.Errorf("expected a failure on unparseable config file")
	}
	reset(cmd)

	_ = cmd.ParseFlags([]string{"-x", "foobar"})
	if ok := Configure("./testdata/config.json", cmd); !ok {
		t.Errorf("unexpected failure calling Configure")
	}

	if excludes, _ := cmd.Flags().GetString("excludes"); excludes != "foobar" {
		t.Errorf("expected 'excludes' to be set to 'foobar', was '%s'", excludes)
	}

	if recurse, _ := cmd.Flags().GetBool("recurse"); recurse != true {
		t.Errorf("expected 'recurse' to be set to true, was %v", recurse)
	}

	if props, _ := cmd.Flags().GetStringSlice("properties"); len(props) != 2 {
		t.Errorf("expected exactly 2 properties, saw %d: %v", len(props), props)
	} else {
		gradlePath := "GradlePath=./gradlew:../gradlew"
		gradleExcludes := "GradleExcludes=node_modules"
		if props[0] != gradlePath && props[0] != gradleExcludes {
			t.Errorf("unexpected property '%s'", props[0])
		}
		if props[1] != gradlePath && props[1] != gradleExcludes {
			t.Errorf("unexpected property '%s'", props[1])
		}
	}

	if filters, _ := cmd.Flags().GetStringSlice("filters"); len(filters) != 2 {
		t.Errorf("expected exactly 2 filters, saw %d: %v", len(filters), filters)
	}
}

func reset(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if slice, ok := flag.Value.(pflag.SliceValue); ok {
			_ = slice.Replace([]string{})
		} else {
			_ = flag.Value.Set("")
		}

		flag.Changed = false
	})
}
