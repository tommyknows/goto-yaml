package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type comment struct {
	text  string
	start token.Pos
	end   token.Pos
}

func main() {
	fset := token.NewFileSet()
	d, err := parser.ParseDir(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	var defaultValues *ast.CompositeLit
	var rootComment string
	types := make(map[string]*ast.TypeSpec)
	var comments []comment
	for _, f := range d {
		for _, ff := range f.Files {
			for _, c := range ff.Comments {
				comments = append(comments, comment{
					text:  c.Text(),
					start: c.Pos(),
					end:   c.End(),
				})
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil {
				return true
			}
			switch x := n.(type) {
			case *ast.ValueSpec:
				if x.Names[0].Name == "DefaultValues" {
					defaultValues = x.Values[0].(*ast.CompositeLit)
					rootComment = x.Doc.Text()
				}
			case *ast.TypeSpec:
				types[x.Name.String()] = x

			}
			return true
		})
	}

	root := &yaml.Node{
		Kind:        yaml.DocumentNode,
		HeadComment: rootComment,
		Content: []*yaml.Node{{
			Kind:    yaml.MappingNode,
			Content: walkValues(defaultValues, types, comments),
		}},
	}

	bytes, err := yaml.Marshal(root)
	if err != nil {
		panic(err)
	}
	fmt.Printf("\n%s\n", bytes)
	os.Exit(0)
}

func walkValues(vals *ast.CompositeLit, types map[string]*ast.TypeSpec, comments []comment) []*yaml.Node {
	var nodes []*yaml.Node

	structType := types[vals.Type.(*ast.Ident).Name].Type.(*ast.StructType)

	var lastVal = vals.Lbrace
	for _, val := range vals.Elts {
		kvExpr, ok := val.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		fieldType := findField(kvExpr.Key.(*ast.Ident).Name, structType.Fields.List)
		comment := fieldType.Doc.Text()
		if c, ok := findComment(kvExpr.Value, lastVal, comments); ok {
			comment += "# \n" + c
		}

		nodes = append(nodes, &yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       getJSONTag(fieldType),
			HeadComment: comment,
		})

		var newNode *yaml.Node
		switch n := kvExpr.Value.(type) {
		case *ast.CompositeLit:
			newNode = &yaml.Node{
				Kind:    yaml.MappingNode,
				Content: walkValues(n, types, comments),
			}
		case *ast.BasicLit:
			newNode = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: strings.Trim(n.Value, "\""),
			}

		case *ast.Ident:
			newNode = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: n.Name,
			}
		}
		if newNode != nil {
			nodes = append(nodes, newNode)
		}
		lastVal = val.End()
	}
	return nodes
}

func findComment(n ast.Expr, lastStatement token.Pos, comments []comment) (string, bool) {
	start := n.Pos()
	for _, c := range comments {
		if c.start < start && c.end < start && c.start > lastStatement {
			return c.text, true
		}
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
