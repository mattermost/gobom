package deps

import (
	"strings"
	"testing"

	"github.com/go-python/gpython/ast"
	"github.com/go-python/gpython/py"
)

func TestVars(t *testing.T) {
	r := strings.NewReader(`
vars = {
	'foo': 'test',
	'bar': True,
	'baz': 'hello'
}`)

	d, err := New(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s := ast.Dump(d.vars["foo"]); s != ast.Dump(&ast.Str{S: "test"}) {
		t.Fatalf("unexpected value: %s", s)
	}
	if s := ast.Dump(d.vars["bar"]); s != ast.Dump(&ast.NameConstant{Value: py.True}) {
		t.Fatalf("unexpected value: %s", s)
	}
	if s := ast.Dump(d.vars["baz"]); s != ast.Dump(&ast.Str{S: "hello"}) {
		t.Fatalf("unexpected value: %s", s)
	}
}

func TestDeps(t *testing.T) {
	r := strings.NewReader(`
vars = {
	'example_git': 'https://git.example',
	'foo_version': '1.0',
	'checkout_bar': True,
	'checkout_baz': False
}

deps = {
	'foo': Var('example_git') + '/foo/foo.git' + '@' + Var('foo_version'),
	'bar': {
		'packages': [
			{
				'package': 'bar/bar/linux-amd64'
			}
		],
		'dep_type': 'cipd',
		'condition': 'checkout_bar and checkout_linux and checkout_amd64'
	},
	'baz': {
		'url': Var('example_git') + '/baz/baz.git' + '@4321',
		'condition': 'checkout_baz'
	}
}`)

	d, err := New(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d.SetTargetOS("linux")
	d.SetTargetCPU("amd64")
	deps := d.Deps()
	if len(deps) != 2 {
		t.Fatalf("unexpected number of deps: %d", len(deps))
	}
	for key, dep := range deps {
		switch key {
		case "foo":
			if dep.Type() != GitDepType {
				t.Fatalf("unexpected dep type")
			}
			if dep.(*GitDep).URL != "https://git.example/foo/foo.git@1.0" {
				t.Fatalf("unexpected dep url: %s", dep.(*GitDep).URL)
			}
		case "bar":
			if dep.Type() != CIPDDepType {
				t.Fatalf("unexpected dep type")
			}
		default:
			t.Fatalf("unexpected key in dep map: %s", key)
		}
	}
}
