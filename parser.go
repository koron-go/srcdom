package srcdom

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

// Parser is a parser for go source files.
type Parser struct {
	Package *Package
}

func (p *Parser) readImport(s *ast.ImportSpec) error {
	path, err := strconv.Unquote(s.Path.Value)
	if err != nil {
		return err
	}
	name := ""
	if s.Name != nil {
		name = s.Name.Name
	}
	p.Package.Imports = append(p.Package.Imports, &Import{
		Name: name,
		Path: path,
	})
	return nil
}

func (p *Parser) readValue(d *ast.GenDecl) error {
	prev := ""
	for _, spec := range d.Specs {
		s, ok := spec.(*ast.ValueSpec)
		if !ok {
			warnf("readValue not support: %T", spec)
			continue
		}
		// determine var/const typeName
		typeName := ""
		var isConst bool
		switch {
		case s.Type == nil:
			if n, imp := baseTypeName(s.Type); !imp {
				typeName = n
			}
		case d.Tok == token.CONST:
			typeName = prev
			isConst = true
		}
		for _, n := range s.Names {
			p.Package.putValue(&Value{
				Name:    n.Name,
				Type:    typeName,
				IsConst: isConst,
			})
		}
	}
	return nil
}

func (p *Parser) readType(spec *ast.TypeSpec) error {
	name := spec.Name.Name
	typ := p.Package.assureType(name)
	typ.defined = true
	return p.readTypeFields(spec.Type, typ)
}

func (p *Parser) readTypeFields(expr ast.Expr, typ *Type) error {
	switch t := expr.(type) {
	case *ast.StructType:
		typ.IsStruct = true
		if t.Fields == nil || len(t.Fields.List) == 0 {
			break
		}
		for _, astField := range t.Fields.List {
			f, err := p.toField(astField)
			if err != nil {
				return err
			}
			if f.Name == "" {
				typ.putEmbedded(f.Type)
				break
			}
			typ.putField(f)
		}
	case *ast.InterfaceType:
		if t.Methods == nil || len(t.Methods.List) == 0 {
			break
		}
		for _, astField := range t.Methods.List {
			name := firstName(astField.Names)
			if name == "" {
				// TODO: support embedded interface
				return errors.New("embedded interface not supported")
			}
			switch ft := astField.Type.(type) {
			case *ast.FuncType:
				f, err := p.toFunc(name, ft)
				if err != nil {
					return err
				}
				typ.putMethod(f)
			default:
				return fmt.Errorf("unsupported interface method type: %T", ft)
			}
		}
	}
	return nil
}

func (p *Parser) readFunc(fun *ast.FuncDecl) error {
	f, err := p.toFunc(fun.Name.Name, fun.Type)
	if err != nil {
		return err
	}
	if fun.Recv != nil {
		if len(fun.Recv.List) == 0 {
			// should not happen (incorrect AST);
			return fmt.Errorf("no receivers: %q", fun.Name.Name)
		}
		recvTypeName, imp := baseTypeName(fun.Recv.List[0].Type)
		if imp {
			// should not happen (incorrect AST);
			return fmt.Errorf("method fro imported receiver: %q", recvTypeName)
		}
		p.Package.assureType(recvTypeName).putMethod(f)
		return nil
	}
	p.Package.putFunc(f)
	return nil
}

func (p *Parser) toFunc(name string, funcType *ast.FuncType) (*Func, error) {
	f := &Func{Name: name}
	if funcType != nil {
		var err error
		f.Params, err = p.toVarArray(funcType.Params)
		if err != nil {
			return nil, err
		}
		f.Results, err = p.toVarArray(funcType.Results)
		if err != nil {
			return nil, err
		}
	}
	return f, nil
}

func (p *Parser) toVarArray(fl *ast.FieldList) ([]*Var, error) {
	if fl == nil || len(fl.List) == 0 {
		return nil, nil
	}
	vars := make([]*Var, 0, len(fl.List))
	for _, f := range fl.List {
		v, err := p.toVar(f)
		if err != nil {
			return nil, err
		}
		vars = append(vars, v)
	}
	return vars, nil
}

func (p *Parser) toVar(f *ast.Field) (*Var, error) {
	return &Var{
		Name: firstName(f.Names),
		Type: typeString(f.Type),
	}, nil
}

func (p *Parser) toField(f *ast.Field) (*Field, error) {
	tag, err := p.toTag(f.Tag)
	if err != nil {
		return nil, err
	}
	return &Field{
		Name: firstName(f.Names),
		Type: typeString(f.Type),
		Tag:  tag,
	}, nil
}

func (p *Parser) toTag(x *ast.BasicLit) (*Tag, error) {
	if x == nil {
		return &Tag{}, nil
	}
	switch x.Kind {
	case token.STRING:
		v, err := strconv.Unquote(x.Value)
		if err != nil {
			return nil, err
		}
		return parseTag(v), nil
	default:
		return nil, fmt.Errorf("unsupported token for tag: %s", x.Kind)
	}
}

// ReadFile read a ast.File to build Package.
func (p *Parser) ReadFile(file *ast.File) error {
	if p.Package == nil || p.Package.Name != file.Name.Name {
		p.Package = &Package{
			Name: file.Name.Name,
		}
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.IMPORT:
				for _, spec := range d.Specs {
					if s, ok := spec.(*ast.ImportSpec); ok {
						err := p.readImport(s)
						if err != nil {
							return err
						}
					}
				}
			case token.CONST, token.VAR:
				err := p.readValue(d)
				if err != nil {
					return err
				}
			case token.TYPE:
				if len(d.Specs) == 1 && !d.Lparen.IsValid() {
					if s, ok := d.Specs[0].(*ast.TypeSpec); ok {
						err := p.readType(s)
						if err != nil {
							return err
						}
					}
					break
				}
				for _, spec := range d.Specs {
					if s, ok := spec.(*ast.TypeSpec); ok {
						err := p.readType(s)
						if err != nil {
							return err
						}
					}
				}
			}
		case *ast.FuncDecl:
			err := p.readFunc(d)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
