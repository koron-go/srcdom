package srcdom

import (
	"fmt"
	"go/ast"
	"strings"
)

// baseTypeName returns the name of the base type of x (or "")
// and whether the type is imported or not.
func baseTypeName(x ast.Expr) (name string, imported bool) {
	switch typ := x.(type) {
	case *ast.Ident:
		return typ.Name, false
	case *ast.SelectorExpr:
		if _, ok := typ.X.(*ast.Ident); ok {
			// only possible for qualified type names;
			// assume type is imported
			return typ.Sel.Name, true
		}
	case *ast.StarExpr:
		return baseTypeName(typ.X)
	}
	return
}

func typeString(x ast.Expr) string {
	switch typ := x.(type) {
	case *ast.Ident:
		return typ.Name
	case *ast.SelectorExpr:
		if _, ok := typ.X.(*ast.Ident); ok {
			return typeString(typ.X) + "." + typ.Sel.Name
		}
	case *ast.StarExpr:
		return "*" + typeString(typ.X)
	case *ast.FuncType:
		fn := toFunc("", typ)
		b := &strings.Builder{}
		b.WriteString("func (" + typesString(fn.Params) + ")")
		fn.writeResults(b)
		return b.String()

	case *ast.Ellipsis:
		return "..." + typeString(typ.Elt)
	case *ast.ArrayType:
		return "[]" + typeString(typ.Elt)
	case *ast.MapType:
		return "map[" + typeString(typ.Key) + "]" + typeString(typ.Value)
	case *ast.StructType:
		if typ.Fields == nil || len(typ.Fields.List) == 0 {
			return "struct{}"
		}
		b := &strings.Builder{}
		b.WriteString("struct { ")
		for i, f := range typ.Fields.List {
			if i > 0 {
				b.WriteString("; ")
			}
			b.WriteString(firstName(f.Names))
			b.WriteString(" ")
			b.WriteString(typeString(f.Type))
		}
		b.WriteString(" }")
		return b.String()

	case *ast.InterfaceType:
		if typ.Methods == nil || len(typ.Methods.List) == 0 {
			return "interface{}"
		}
		b := &strings.Builder{}
		b.WriteString("interface { ")
		for i, m := range typ.Methods.List {
			if i > 0 {
				b.WriteString("; ")
			}
			switch mTyp := m.Type.(type) {
			case *ast.FuncType:
				fn := toFunc("", mTyp)
				b.WriteString(firstName(m.Names))
				b.WriteString("(" + typesString(fn.Params) + ")")
				fn.writeResults(b)
			default:
				panic(fmt.Sprintf("not supported fields in unnamed interface type: %T", m.Type))
			}
		}
		b.WriteString(" }")
		return b.String()

	case *ast.ChanType:
		var chanLabel string
		switch typ.Dir {
		case ast.SEND | ast.RECV:
			chanLabel = "chan"
		case ast.SEND:
			chanLabel = "chan<-"
		case ast.RECV:
			chanLabel = "<-chan"
		default:
			panic(fmt.Sprintf("illegal channel direction (ast.ChanDir): %d", typ.Dir))
		}
		return chanLabel + " " + typeString(typ.Value)

	default:
		warnf("typeString doesn't support: %T", typ)
	}
	return ""
}

func firstName(names []*ast.Ident) string {
	if len(names) == 0 {
		return ""
	}
	return names[0].Name
}

func toVar(f *ast.Field) []*Var {
	typ := typeString(f.Type)
	if len(f.Names) == 0 {
		return []*Var{{Name: "", Type: typ}}
	}
	vars := make([]*Var, len(f.Names))
	for i, n := range f.Names {
		vars[i] = &Var{Name: n.Name, Type: typ}
	}
	return vars
}

func toVarArray(fl *ast.FieldList) []*Var {
	if fl == nil || len(fl.List) == 0 {
		return nil
	}
	vars := make([]*Var, 0, len(fl.List))
	for _, f := range fl.List {
		vars = append(vars, toVar(f)...)
	}
	return vars
}

func typesString(vars []*Var) string {
	if len(vars) == 0 {
		return ""
	}
	b := &strings.Builder{}
	for i, v := range vars {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(v.Type)
	}
	return b.String()
}

func toFunc(name string, funcType *ast.FuncType) *Func {
	f := &Func{Name: name}
	if funcType != nil {
		f.Params = toVarArray(funcType.Params)
		f.Results = toVarArray(funcType.Results)
	}
	return f
}
