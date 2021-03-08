package gradle

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/mattermost/gobom"
)

func TestUnmarshal(t *testing.T) {
	b, err := ioutil.ReadFile("./testdata/gradle-dependencies.txt")
	if err != nil {
		t.Fatal(err)
	}

	reader := bytes.NewReader(b)
	nodes := unmarshal(reader)
	trees := []*buildConfig{}
	for _, node := range nodes {
		if len(node.nodes) > 0 {
			trees = append(trees, (*buildConfig)(node))
		}
	}

	if len(trees) != 19 {
		t.Fatalf("wrong number of trees: expected %d, saw %d", 19, len(trees))
	}
}

func TestParseDependency(t *testing.T) {
	values := map[string]string{
		"com.wix:detox:+ -> 18.1.1":                                                         "false|com.wix|detox|18.1.1|true",
		"org.hamcrest:hamcrest-library:1.3 (*)":                                             "false|org.hamcrest|hamcrest-library|1.3|true",
		"project react-native-local-auth (n)":                                               "true||react-native-local-auth|unknown|false",
		"com.android.support:appcompat-v7:28.0.0 -> androidx.appcompat:appcompat:1.1.0 (*)": "false|androidx.appcompat|appcompat|1.1.0|true",
	}
	for value, expected := range values {
		dependency := dependency{value: value + "\n"}
		parsed := dependency.Parse()
		if fmt.Sprintf("%t|%s|%s|%s|%t", parsed.project, parsed.Group, parsed.Name, parsed.Version, parsed.resolved) != expected {
			t.Errorf("unexpected parser output for dependency '%s': '%v'", value, dependency.Parse())
		}
	}
}

func TestGenerateBOM(t *testing.T) {
	g := Generator{}
	g.Configure(gobom.Options{Recurse: true})
	g.GradlePath = []string{"./gradlew"}

	bom, err := g.GenerateBOM("./testdata/testproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := 0
	expected := map[string]string{
		"compileClasspath":     "",
		"runtimeClasspath":     "",
		"testCompileClasspath": "",
		"testRuntimeClasspath": "",
		"joda-time":            "pkg:maven/joda-time/joda-time@2.2",
		"junit":                "pkg:maven/junit/junit@4.12",
		"hamcrest-core":        "pkg:maven/org.hamcrest/hamcrest-core@1.3",
	}
	for _, component := range bom.Components {
		if purl, ok := expected[component.Name]; ok && purl == component.PURL {
			found++
		} else if ok && purl != component.PURL {
			t.Errorf("unexpected purl: expected '%s', saw '%s'", purl, component.PURL)
		} else {
			t.Errorf("unexpected component: '%s'", component.Name)
		}
	}
	if found != len(expected) {
		t.Errorf("expected %d components, saw %d", len(expected), found)
	}
}
