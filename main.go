package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/tools/go/loader"
	"gopkg.in/yaml.v3"
)

type comment struct {
	text  string
	start token.Pos
	end   token.Pos
}

func main() {
	pkgPath := os.Args[1]
	conf := loader.Config{ParserMode: parser.ParseComments}
	conf.Import(pkgPath)
	loadedPackages, err := conf.Load()
	if err != nil {
		panic(err)
	}

	packages := make(map[string]*Package)
	for _, pkg := range loadedPackages.AllPackages {
		packages[pkg.Pkg.Path()] = walkPkg(pkg, pkg.Pkg.Path())
	}

	var defaultValue ast.Expr
	var rootComment string
	if v, ok := packages[pkgPath].variables[pkgPath+"/DefaultValues"]; ok {
		defaultValue = v.Values[0]
		rootComment = v.Doc.Text()
	}

	root := &yaml.Node{
		Kind:        yaml.DocumentNode,
		HeadComment: rootComment,
		Content: []*yaml.Node{
			walkExpr(defaultValue, packages[pkgPath], packages),
		},
	}

	bytes, err := yaml.Marshal(root)
	if err != nil {
		panic(err)
	}
	fmt.Printf("\n%s\n", bytes)
	os.Exit(0)
}

func walkPkg(pkg *loader.PackageInfo, importPath string) *Package {
	p := &Package{
		path:      importPath,
		comments:  make([]comment, 0),
		imports:   make(map[string]string),
		variables: make(map[string]*ast.ValueSpec),
		types:     make(map[string]*ast.TypeSpec),
		files:     pkg.Files,
	}

	for _, f := range pkg.Files {
		for _, c := range f.Comments {
			p.comments = append(p.comments, comment{
				text:  c.Text(),
				start: c.Pos(),
				end:   c.End(),
			})
		}

		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil {
				return true
			}
			switch x := n.(type) {
			case *ast.ValueSpec:
				p.variables[importPath+"/"+x.Names[0].Name] = x

			case *ast.ImportSpec:
				if x.Name != nil {
					p.imports[x.Name.Name] = strings.Trim(x.Path.Value, "\"")
				} else {
					p.imports[path.Base(strings.Trim(x.Path.Value, "\""))] = strings.Trim(x.Path.Value, "\"")
				}

			case *ast.TypeSpec:
				p.types[x.Name.String()] = x

			default:
				//fmt.Printf("unhandled type %T\n", x)
			}
			return true
		})
	}
	sort.Slice(p.comments, func(i, j int) bool {
		return p.comments[i].end < p.comments[j].end
	})

	return p
}

func pkgImportPath(sel *ast.SelectorExpr) string {
	return sel.X.(*ast.Ident).Name
}

type Package struct {
	types     map[string]*ast.TypeSpec
	variables map[string]*ast.ValueSpec
	imports   map[string]string
	comments  []comment
	path      string
	files     []*ast.File
}

func walkCompositeLit(
	val *ast.CompositeLit,
	p *Package,
	packages map[string]*Package,
) *yaml.Node {
	switch t := val.Type.(type) {
	case *ast.Ident:
		switch ts := p.types[t.Name].Type.(type) {
		case *ast.StructType:
			return walkStruct(val, ts, p, packages)
		default:
			panic(fmt.Sprintf("unhandled ident type: %T", ts))
		}

	case *ast.SelectorExpr:
		// TODO: deduplicate with ast.Ident?
		switch ts := packages[p.imports[pkgImportPath(t)]].types[t.Sel.Name].Type.(type) {
		case *ast.StructType:
			return walkStruct(val, ts, p, packages)
		default:
			panic(fmt.Sprintf("unhandled ident type: %T", ts))
		}

	case *ast.ArrayType:
		return walkArray(val, p, packages)

	case *ast.MapType:
		return walkMap(val, p, packages)

	default:
		panic(fmt.Sprintf("unhandled identifier %T", t))
	}

}

func walkMap(vals *ast.CompositeLit, p *Package, packages map[string]*Package) *yaml.Node {
	root := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{},
	}

	for _, e := range vals.Elts {
		kvExpr, ok := e.(*ast.KeyValueExpr)
		if !ok {
			panic(fmt.Sprintf("unhandled map type: %T", e))
		}
		key := walkExpr(kvExpr.Key, p, packages)
		// we don't need to be concerned about field-type comments as this is a map.
		key.HeadComment, _ = findComment(kvExpr.Key, p.comments)
		root.Content = append(root.Content, key, walkExpr(kvExpr.Value, p, packages))
	}
	return root
}

func walkArray(vals *ast.CompositeLit, p *Package, packages map[string]*Package) *yaml.Node {
	root := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: []*yaml.Node{},
	}

	for _, e := range vals.Elts {
		if newNode := walkExpr(e, p, packages); newNode != nil {
			if c, ok := findComment(e, p.comments); ok {
				newNode.HeadComment += c
			}
			root.Content = append(root.Content, newNode)
		}
	}

	return root
}
func walkExpr(e ast.Expr, p *Package, packages map[string]*Package) *yaml.Node {
	switch n := e.(type) {
	case *ast.CompositeLit:
		return walkCompositeLit(n, p, packages)

	case *ast.BasicLit:
		return walkBasicLit(n)

	case *ast.Ident:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: n.Name,
		}

	case *ast.KeyValueExpr:
		panic("refusing to walk key-value expression as this would result in two nodes.")

	default:
		panic(fmt.Sprintf("unhandled identifier %T", n))
	}
}

func walkStruct(vals *ast.CompositeLit, structType *ast.StructType, p *Package, packages map[string]*Package) *yaml.Node {
	root := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{},
	}
	for _, val := range vals.Elts {
		kvExpr, ok := val.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		field := findField(kvExpr.Key.(*ast.Ident).Name, structType.Fields.List)
		var docs []string
		switch ft := field.Type.(type) {
		case *ast.SelectorExpr:
			pkg := packages[p.imports[ft.X.(*ast.Ident).Name]]
			if ts, ok := pkg.types[ft.Sel.Name]; ok {
				if c, ok := findComment(ts.Name, pkg.comments); ok {
					docs = append(docs, c)
				}
			}
		case *ast.Ident:
			switch ft.Name {
			case "bool", "string", "int":
				// nothing to do, these types cannot be commented
			default:
				// check if a type exists.
				if ts, ok := p.types[ft.Name]; ok {
					if c, ok := findComment(ts.Name, p.comments); ok {
						docs = append(docs, c)
					}
				}
			}
		}
		if c := field.Doc.Text(); c != "" {
			docs = append(docs, c)
		}
		if c, ok := findComment(kvExpr.Value, p.comments); ok {
			docs = append(docs, c)
		}

		root.Content = append(root.Content, &yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       getJSONTag(field),
			HeadComment: toYAMLComment(docs),
		})

		if newNode := walkExpr(kvExpr.Value, p, packages); newNode != nil {
			root.Content = append(root.Content, newNode)
		}
	}
	return root
}

func findTypeComment(t ast.Expr, p *Package, packages map[string]*Package) (string, bool) {
	switch ft := t.(type) {
	case *ast.SelectorExpr:
		pkg := packages[p.imports[ft.X.(*ast.Ident).Name]]
		return findComment(pkg.types[ft.Sel.Name].Name, pkg.comments)

	case *ast.Ident:
		switch ft.Name {
		case "bool", "string", "int":
			// nothing to do, these types cannot be commented
			return "", false
		default:
			// TODO: could it happen that we don't find the type?
			// same for the selector above.
			return findComment(p.types[ft.Name].Name, p.comments)
		}
	default:
		panic(fmt.Sprintf("unhandled type %T", ft))
	}
}

func toYAMLComment(comments []string) string {
	return strings.Join(comments, "#\n")
}

func walkBasicLit(n *ast.BasicLit) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: strings.Trim(n.Value, "\""),
	}
}

// TODO: does / could this produce false positives?
func findComment(n ast.Expr, comments []comment) (string, bool) {
	start := n.Pos()
	var closest comment
	for _, c := range comments {
		if c.start < start && c.end < start && (closest.text == "" || closest.end < c.start) {
			closest = c
		}
	}
	if closest.text != "" {
		return closest.text, true
	}
	return "", false
}

func getJSONTag(f *ast.Field) string {
	return reflect.StructTag(strings.Trim(
		f.Tag.Value, "`",
	)).Get("json")
}

func findField(name string, in []*ast.Field) *ast.Field {
	for _, f := range in {
		if f.Names[0].Name == name {
			return f
		}
	}
	return nil
}
