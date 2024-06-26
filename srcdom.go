/*
Package srcdom provides utilities to manipulate Go's AST (Abstract Structure
Tree). Using srcdom, you can easily access/extract information of types,
funcions and variables from AST.
*/
package srcdom

import (
	"go/ast"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func isPublicName(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return false
	}
	return unicode.IsUpper(r)
}

func sortedNames(m map[string]int) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Package represents a go package.
type Package struct {
	Name string

	Imports []*Import

	Values []*Value
	valIdx map[string]int

	Funcs  []*Func
	funIdx map[string]int

	Types  []*Type
	typIdx map[string]int
}

func (p *Package) putValue(v *Value) {
	if p.valIdx == nil {
		p.valIdx = make(map[string]int)
	}
	idx := len(p.Values)
	p.valIdx[v.Name] = idx
	p.Values = append(p.Values, v)
}

// Value gets a value which matches with name.
func (p *Package) Value(name string) (*Value, bool) {
	idx, ok := p.valIdx[name]
	if !ok {
		return nil, false
	}
	return p.Values[idx], true
}

// ValueName returns sorted names of value in the package.
func (p *Package) ValueNames() []string {
	return sortedNames(p.valIdx)
}

func (p *Package) putFunc(fun *Func) {
	if p.funIdx == nil {
		p.funIdx = make(map[string]int)
	}
	idx := len(p.Funcs)
	p.funIdx[fun.Name] = idx
	p.Funcs = append(p.Funcs, fun)
}

// Func gets a func which matches with name.
func (p *Package) Func(name string) (*Func, bool) {
	idx, ok := p.funIdx[name]
	if !ok {
		return nil, false
	}
	return p.Funcs[idx], true
}

// FuncNames returns sorted names of function in the package.
func (p *Package) FuncNames() []string {
	return sortedNames(p.funIdx)
}

func (p *Package) assureType(name string) *Type {
	if typ, ok := p.Type(name); ok {
		return typ
	}
	typ := &Type{Name: name}
	p.putType(typ)
	return typ
}

func (p *Package) putType(typ *Type) {
	if p.typIdx == nil {
		p.typIdx = make(map[string]int)
	}
	idx := len(p.Types)
	p.typIdx[typ.Name] = idx
	p.Types = append(p.Types, typ)
}

// Type gets a type which matches with name.
func (p *Package) Type(name string) (*Type, bool) {
	idx, ok := p.typIdx[name]
	if !ok {
		return nil, false
	}
	return p.Types[idx], true
}

// TypeNames returns sorted names of type in the package.
func (p *Package) TypeNames() []string {
	return sortedNames(p.typIdx)
}

// Import represents an import.
type Import struct {
	Name string
	Path string
}

// Var represents a variable.
type Var struct {
	Name string
	Type string
}

// Field represents a variable.
type Field struct {
	Name string
	Type string
	Tag  *Tag
}

// Tag represents a tag for field
type Tag struct {
	Raw string

	Values   []*TagValue
	valueIdx map[string]int
}

// TagValue gets a tag value which matches with name.
func (tag *Tag) TagValue(n string) (*TagValue, bool) {
	idx, ok := tag.valueIdx[n]
	if !ok {
		return nil, false
	}
	return tag.Values[idx], false
}

func (tag *Tag) putTagValue(v *TagValue) {
	if tag.valueIdx == nil {
		tag.valueIdx = make(map[string]int)
	}
	idx := len(tag.Values)
	tag.valueIdx[v.Name] = idx
	tag.Values = append(tag.Values, v)
}

func parseTag(tag string) *Tag {
	dst := &Tag{Raw: tag}
	// parse tag: partially copied from reflect.StructTag.Lookup()
	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Scan to colon.
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// Scan quoted string to find value.
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		value, err := strconv.Unquote(qvalue)
		if err != nil {
			break
		}
		dst.putTagValue(parseTagValue(name, value))
	}
	return dst
}

func (tag *Tag) match(name string, value *string) bool {
	for _, v := range tag.Values {
		if v.Name == name {
			if value == nil || v.has(*value) {
				return true
			}
		}
	}
	return false
}

var tagValueRx = regexp.MustCompile(`\s+`)

// TagValue represents content of a tag.
type TagValue struct {
	Name   string
	Raw    string
	Values []string
}

func parseTagValue(name, s string) *TagValue {
	return &TagValue{
		Name:   name,
		Raw:    s,
		Values: tagValueRx.Split(s, -1),
	}
}

func (tv *TagValue) has(value string) bool {
	for _, v := range tv.Values {
		if v == value {
			return true
		}
	}
	return false
}

// Func represents a function.
type Func struct {
	Name    string
	Params  []*Var
	Results []*Var
}

// IsPublic checks its name is public or not.
func (fn *Func) IsPublic() bool {
	return isPublicName(fn.Name)
}

func (fn *Func) writeResults(b *strings.Builder) {
	rets := typesString(fn.Results)
	switch len(fn.Results) {
	case 0:
		// nothing to append
	case 1:
		b.WriteString(" ")
		b.WriteString(rets)
	default:
		b.WriteString(" (")
		b.WriteString(rets)
		b.WriteString(")")
	}
}

// Type represents a function.
type Type struct {
	Name    string
	Defined bool

	IsStruct    bool
	IsInterface bool

	Embeds   []string
	embedIdx map[string]int

	Fields   []*Field
	fieldIdx map[string]int

	Methods   []*Func
	methodIdx map[string]int
}

func (typ *Type) putEmbed(typeName string) {
	if typ.embedIdx == nil {
		typ.embedIdx = make(map[string]int)
	}
	idx := len(typ.Embeds)
	typ.embedIdx[typeName] = idx
	typ.Embeds = append(typ.Embeds, typeName)
}

// IsPublic checks its name is public or not.
func (typ *Type) IsPublic() bool {
	return isPublicName(typ.Name)
}

// Embed checks the type has embed type or not.
func (typ *Type) Embed(n string) bool {
	_, ok := typ.embedIdx[n]
	return ok
}

func (typ *Type) putField(f *Field) {
	if typ.fieldIdx == nil {
		typ.fieldIdx = make(map[string]int)
	}
	idx := len(typ.Fields)
	typ.fieldIdx[f.Name] = idx
	typ.Fields = append(typ.Fields, f)
}

// Field gets a field which matches with name.
func (typ *Type) Field(n string) (*Field, bool) {
	idx, ok := typ.fieldIdx[n]
	if !ok {
		return nil, false
	}
	return typ.Fields[idx], true
}

func (typ *Type) putMethod(fun *Func) {
	if typ.methodIdx == nil {
		typ.methodIdx = make(map[string]int)
	}
	idx := len(typ.Methods)
	typ.methodIdx[fun.Name] = idx
	typ.Methods = append(typ.Methods, fun)
}

// Method gets a method Func which matches with name.
func (typ *Type) Method(n string) (*Func, bool) {
	idx, ok := typ.methodIdx[n]
	if !ok {
		return nil, false
	}
	return typ.Methods[idx], true
}

// FieldsByTag collects fields which match with query.
// The query's format is "{tagName}" or "{tagName}:{value}".
func (typ *Type) FieldsByTag(tagQuery string) []*Field {
	var hits []*Field
	var name string
	var value *string
	for _, f := range typ.Fields {
		if f.Tag != nil && f.Tag.match(name, value) {
			hits = append(hits, f)
		}
	}
	return hits
}

// Value represents a value or const
type Value struct {
	Name    string
	Type    string
	IsConst bool

	Literal *ast.BasicLit
}

// IsPublic checks its name is public or not.
func (v *Value) IsPublic() bool {
	return isPublicName(v.Name)
}
