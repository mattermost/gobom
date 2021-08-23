package deps

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-python/gpython/ast"
	"github.com/go-python/gpython/parser"
	"github.com/go-python/gpython/py"
)

type DEPS struct {
	*http.Client

	path        string
	vars        map[string]ast.Expr
	deps        map[string]ast.Expr
	relative    bool
	recursedeps []ast.Expr
	resolved    map[string]*DEPS
	errors      []error
}

// New parses a DEPS definition
func New(r io.Reader) (*DEPS, error) {
	d := &DEPS{path: ".", Client: http.DefaultClient}
	module, err := parser.Parse(r, "DEPS", "exec")
	if err != nil {
		return nil, err
	}
	body := module.(*ast.Module).Body
	for _, statement := range body {
		if statement.Type() == ast.AssignType {
			assign := statement.(*ast.Assign)
			for _, target := range assign.Targets {
				switch target.(*ast.Name).Id {
				case "vars":
					d.vars, err = dictToMap(assign.Value)
					if err != nil {
						return d, fmt.Errorf("error parsiong vars: %v", err)
					}
				case "deps":
					d.deps, err = dictToMap(assign.Value)
					if err != nil {
						return d, fmt.Errorf("error parsing deps dict: %v", err)
					}
				case "use_relative_paths":
					relative, ok := assign.Value.(*ast.NameConstant)
					if !ok {
						return d, fmt.Errorf("error parsing boolean use_relative_paths")
					}
					d.relative = relative.Value == py.True
				case "recursedeps":
					recursedeps, ok := assign.Value.(*ast.List)
					if !ok {
						err := fmt.Errorf("error parsing recursedeps: expected a list, saw %v", assign.Value.Type())
						return d, err
					}
					d.recursedeps = recursedeps.Elts
				}
			}
		}
	}

	if d.vars == nil {
		d.vars = make(map[string]ast.Expr)
	}

	return d, nil
}

func (d *DEPS) SetTargetOS(os ...string) {
	for _, target := range os {
		d.setBoolDefault("checkout_"+target, true)
	}
	for _, target := range []string{"android", "chromeos", "fuchsia", "ios", "linux", "mac", "win"} {
		d.setBoolDefault("checkout_"+target, false)
	}
}

func (d *DEPS) SetTargetCPU(os ...string) {
	for _, target := range os {
		d.setBoolDefault("checkout_"+target, true)
	}
	for _, target := range []string{"arm", "arm64", "x86", "mips", "mips64", "ppc", "s390", "x64"} {
		d.setBoolDefault("checkout_"+target, false)
	}
}

func (d *DEPS) SetHostOS(os string) {
	d.setStringDefault("host_os", os)
}

func (d *DEPS) SetHostCPU(cpu string) {
	d.setStringDefault("host_cpu", cpu)
}

func (d *DEPS) setVars(vars map[string]ast.Expr) {
	for name, value := range vars {
		d.vars[name] = value
	}
}

func (d *DEPS) SetBoolVar(name string, value bool) {
	d.vars[name] = &ast.NameConstant{Value: py.Bool(value)}
}

func (d *DEPS) SetStringVar(name string, value string) {
	d.vars[name] = &ast.Str{S: py.String(value)}
}

func (d *DEPS) setBoolDefault(name string, value bool) {
	if d.vars[name] == nil {
		d.vars[name] = &ast.NameConstant{Value: py.Bool(value)}
	}
}

func (d *DEPS) setStringDefault(name string, value string) {
	if d.vars[name] == nil {
		d.vars[name] = &ast.Str{S: py.String(value)}
	}
}

func (d *DEPS) Deps() map[string]Dep {
	result := make(map[string]Dep)
	for path, depExpr := range d.deps {
		var dep Dep
		if dict, ok := depExpr.(*ast.Dict); ok {
			depmap, _ := dictToMap(dict)
			url, _ := d.resolveStrExpr(depmap["url"])
			typ, err := d.resolveStrExpr(depmap["dep_type"])
			if err != nil || typ == "git" {
				dep = &GitDep{URL: url, Parent: d.path}
			} else {
				dep = &CIPDDep{Parent: d.path}
			}
			condition, err := d.resolveConditionExpr(depmap["condition"])
			if err != nil {
				// TODO: proper error handling
				fmt.Println(depmap["condition"].(*ast.Str).S, condition, err)
			}
			if !condition {
				continue
			}
		} else {
			url, err := d.resolveStrExpr(depExpr)
			if err != nil {
				continue
			}
			dep = &GitDep{URL: url, Parent: d.path}
		}
		if _, ok := result[path]; ok {
			// TODO: handle more gracefully
			panic("conflict on " + path)
		}
		result[path] = dep
	}
	for root, resolved := range d.resolved {
		r := resolved.Deps()
		for path, dep := range r {
			if d.relative {
				path = filepath.Join(root, path)
			}
			if _, ok := result[path]; ok {
				// TODO: handle more gracefully
				panic("conflict on " + path)
			}
			result[path] = dep
		}
	}
	return result
}

func (d *DEPS) Errors() []error {
	return d.errors
}

func (d *DEPS) Resolve() error {
	if d.resolved != nil {
		return nil
	}

	d.resolved = make(map[string]*DEPS, len(d.recursedeps))
	for _, keyExpr := range d.recursedeps {
		key, err := d.resolveStrExpr(keyExpr)
		if err != nil {
			err = fmt.Errorf("error parsing recursedeps: %v", err)
			d.errors = append(d.errors, err)
			return err
		}
		if depExpr, ok := d.deps[key]; ok {
			url := ""
			if dep, ok := depExpr.(*ast.Dict); ok {
				dict, _ := dictToMap(dep)
				condition, err := d.resolveConditionExpr(dict["condition"])
				if err != nil {
					// TODO: proper error handling
					fmt.Println(dict["condition"].(*ast.Str).S, condition, err)
				}
				if !condition {
					continue
				}
				url, err = d.resolveStrExpr(dict["url"])
				if err != nil {
					d.errors = append(d.errors, err)
					return err
				}
			} else {
				url, err = d.resolveStrExpr(depExpr)
				if err != nil {
					d.errors = append(d.errors, err)
					return err
				}
			}
			if resolved, err := d.fetchDEPS(url); err == nil {
				resolved.path = key
				d.resolved[key] = resolved
			} else {
				d.errors = append(d.errors, err)
			}
		}
	}

	for _, resolved := range d.resolved {
		resolved.setVars(d.vars)
		err := resolved.Resolve()
		if err != nil {
			d.errors = append(d.errors, resolved.errors...)
		}
	}

	if len(d.errors) != 0 {
		return fmt.Errorf("encountered %d errors during resolve", len(d.errors))
	}

	return nil
}

var (
	googlesourceRegexp = regexp.MustCompile(`^https://[A-Za-z0-9\-]*\.googlesource\.com/`)
	githubRegexp       = regexp.MustCompile(`^https://github\.com/([^/]*)/([^@]*)\.git@(.*)$`)
)

func (d *DEPS) fetchDEPS(url string) (*DEPS, error) {
	switch {
	case googlesourceRegexp.MatchString(url):
		url = strings.Replace(url, "@", "/+/", 1) + "/DEPS?format=TEXT"
		response, err := d.Get(url)
		defer response.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch DEPS from '%s': %v", url, err)
		}
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch DEPS from '%s': %s", url, response.Status)
		}
		decoder := base64.NewDecoder(base64.StdEncoding, response.Body)
		return New(decoder)
	case githubRegexp.MatchString(url):
		matches := githubRegexp.FindStringSubmatch(url)
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/DEPS?ref=%s", matches[1], matches[2], matches[3])
		response, err := d.Get(url)
		defer response.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch DEPS from '%s': %v", url, err)
		}
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch DEPS from '%s': %s", url, response.Status)
		}
		jsonDecoder := json.NewDecoder(response.Body)
		data := &struct {
			Content string
		}{}
		err = jsonDecoder.Decode(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode response from GitHub: %v", err)
		}
		decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(data.Content))
		return New(decoder)
	default:
		return nil, fmt.Errorf("unsupported remote, cannot fetch: %v", url)
	}
}

func (d *DEPS) resolveConditionExpr(expr ast.Expr) (bool, error) {
	if expr == nil {
		// nil condition, i.e. unconditionally true
		return true, nil
	}
	switch expr.Type() {
	case ast.StrType:
		expression, err := parser.ParseString(string(expr.(*ast.Str).S), "eval")
		if err != nil {
			return false, err
		}
		return d.resolveConditionExpr(expression.(*ast.Expression).Body)
	case ast.NameType:
		name := string(expr.(*ast.Name).Id)
		value := d.vars[name]
		if value == nil {
			return false, fmt.Errorf("undeclared variable: %s", name)
		}
		return d.resolveConditionExpr(value)
	case ast.NameConstantType:
		nc := expr.(*ast.NameConstant)
		return nc.Value == py.True, nil
	case ast.BoolOpType:
		op := expr.(*ast.BoolOp)
		if op.Op == ast.And {
			for _, val := range op.Values {
				b, err := d.resolveConditionExpr(val)
				if !b || err != nil {
					return false, err
				}
			}
			return true, nil
		} else {
			for _, val := range op.Values {
				b, err := d.resolveConditionExpr(val)
				if b || err != nil {
					return b, err
				}
			}
			return false, nil
		}
	case ast.UnaryOpType:
		op := expr.(*ast.UnaryOp)
		if op.Op != ast.Not {
			return false, fmt.Errorf("unsupported unary op: %s", op.Op.String())
		}
		b, err := d.resolveConditionExpr(op.Operand)
		return !b, err
	case ast.CompareType:
		compare := expr.(*ast.Compare)
		if len(compare.Comparators) != 1 || len(compare.Ops) != 1 || compare.Ops[0] != ast.Eq {
			return false, fmt.Errorf("unsupported comparison")
		}
		left, err := d.resolveStrExpr(compare.Left)
		if err != nil {
			return false, err
		}
		right, err := d.resolveStrExpr(compare.Comparators[0])
		if err != nil {
			return false, err
		}
		return left == right, nil
	default:
		return false, fmt.Errorf("unsupported expression type: %v", expr.Type())
	}
}

func (d *DEPS) resolveStrExpr(expr ast.Expr) (string, error) {
	str := ""
	if expr == nil {
		return str, fmt.Errorf("unable to resolve nil expression to string")
	}
	switch expr.Type() {
	case ast.StrType:
		return regexp.MustCompile(`\{[^}]+\}`).ReplaceAllStringFunc(
			string(expr.(*ast.Str).S),
			func(s string) string {
				if val, ok := d.vars[s[1:len(s)-1]]; ok {
					replacement, err := d.resolveStrExpr(val)
					if err == nil {
						return replacement
					}
				}
				return s
			}), nil
	case ast.BinOpType:
		binOp := expr.(*ast.BinOp)
		switch binOp.Op {
		case ast.Add:
			left, err := d.resolveStrExpr(binOp.Left)
			if err != nil {
				return left, err
			}
			right, err := d.resolveStrExpr(binOp.Right)
			return left + right, err
		default:
			return str, fmt.Errorf("unsupported binary op: %v", binOp.Op.String())
		}
	case ast.CallType:
		call := expr.(*ast.Call)
		name, ok := call.Func.(*ast.Name)
		if !ok {
			return "", fmt.Errorf("unsupported call expression")
		}
		switch string(name.Id) {
		case "Var":
			if len(call.Args) != 1 {
				return "", fmt.Errorf("unexpected number of arguments in call to Var")
			}
			varname, err := d.resolveStrExpr(call.Args[0])
			if err != nil {
				return "", err
			}
			return d.resolveStrExpr(d.vars[varname])
		default:
			return "", fmt.Errorf("unsupported function name: %v", name.Id)
		}
	case ast.NameType:
		name := string(expr.(*ast.Name).Id)
		return d.resolveStrExpr(d.vars[name])
	default:
		return str, fmt.Errorf("unsupported expression type: %v", expr.Type())
	}
}

func dictToMap(expr ast.Expr) (map[string]ast.Expr, error) {
	out := make(map[string]ast.Expr)
	if expr == nil {
		return out, nil
	}
	dict, ok := expr.(*ast.Dict)
	if !ok {
		return nil, fmt.Errorf("dict expected, saw %v", expr.Type())
	}
	for i, keyExpr := range dict.Keys {
		key, ok := keyExpr.(*ast.Str)
		if !ok {
			return nil, fmt.Errorf("dict key not a string")
		}
		out[string(key.S)] = dict.Values[i]
	}
	return out, nil
}
