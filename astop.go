package srcdom

import "go/ast"

// baseTypeName returns the name of the base type of x (or "")
// and whether the type is imported or not.
func baseTypeName(x ast.Expr) (name string, imported bool) {
	switch t := x.(type) {
	case *ast.Ident:
		return t.Name, false
	case *ast.SelectorExpr:
		if _, ok := t.X.(*ast.Ident); ok {
			// only possible for qualified type names;
			// assume type is imported
			return t.Sel.Name, true
		}
	case *ast.StarExpr:
		return baseTypeName(t.X)
	}
	return
}

func typeString(x ast.Expr) string {
	switch t := x.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if _, ok := t.X.(*ast.Ident); ok {
			return typeString(t.X) + "." + t.Sel.Name
		}
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	}
	return ""
}

func firstName(names []*ast.Ident) string {
	if len(names) == 0 {
		return ""
	}
	return names[0].Name
}
