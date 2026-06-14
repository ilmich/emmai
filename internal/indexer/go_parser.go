package indexer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// parseGoFile extracts symbols from a Go source file using go/ast.
// Returns symbols, package name, and any parse error.
func parseGoFile(absPath, relPath string) ([]Symbol, string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, absPath, nil, 0)
	if err != nil {
		return nil, "", err
	}

	pkg := f.Name.Name
	var symbols []Symbol

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			sym := Symbol{
				File:      relPath,
				Line:      fset.Position(d.Pos()).Line,
				Package:   pkg,
				Signature: renderFieldList(d.Type.Params) + " " + renderFieldList(d.Type.Results),
			}
			if d.Recv != nil && len(d.Recv.List) > 0 {
				sym.Kind = "method"
				sym.Receiver = renderExpr(d.Recv.List[0].Type)
				sym.Name = d.Name.Name
			} else {
				sym.Kind = "func"
				sym.Name = d.Name.Name
			}
			symbols = append(symbols, sym)

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					kind := "type"
					switch s.Type.(type) {
					case *ast.StructType:
						kind = "struct"
					case *ast.InterfaceType:
						kind = "interface"
					}
					symbols = append(symbols, Symbol{
						Name:    s.Name.Name,
						Kind:    kind,
						File:    relPath,
						Line:    fset.Position(s.Pos()).Line,
						Package: pkg,
					})
				case *ast.ValueSpec:
					kind := "var"
					if d.Tok == token.CONST {
						kind = "const"
					}
					for _, name := range s.Names {
						symbols = append(symbols, Symbol{
							Name:    name.Name,
							Kind:    kind,
							File:    relPath,
							Line:    fset.Position(name.Pos()).Line,
							Package: pkg,
						})
					}
				}
			}
		}
	}

	return symbols, pkg, nil
}

func renderFieldList(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}
	var parts []string
	for _, f := range fl.List {
		typ := renderExpr(f.Type)
		if len(f.Names) == 0 {
			parts = append(parts, typ)
		} else {
			var names []string
			for _, n := range f.Names {
				names = append(names, n.Name)
			}
			parts = append(parts, strings.Join(names, ", ")+" "+typ)
		}
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func renderExpr(e ast.Expr) string {
	if e == nil {
		return ""
	}
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + renderExpr(t.X)
	case *ast.SelectorExpr:
		return renderExpr(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + renderExpr(t.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", renderExpr(t.Key), renderExpr(t.Value))
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + renderExpr(t.Elt)
	case *ast.ChanType:
		return "chan " + renderExpr(t.Value)
	default:
		return "..."
	}
}
