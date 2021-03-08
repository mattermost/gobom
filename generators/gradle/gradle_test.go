package gradle

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
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
