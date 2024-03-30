package srcdom

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"log"
	"os"
	"sort"
)

// readFile reads a file as a Package.
func readFile(name string) (*Package, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, f, 0)
	if err != nil {
		return nil, err
	}
	p := &Parser{}
	err = p.ScanFile(file)
	if err != nil {
		return nil, err
	}
	return p.Package, nil
}

func sortFileNames(src map[string]*ast.File) []string {
	names := make([]string, 0, len(src))
	for n := range src {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i] < names[j]
	})
	return names
}

func toPackages(pkgMap map[string]*ast.Package) []*ast.Package {
	pkgs := make([]*ast.Package, 0, len(pkgMap))
	for _, p := range pkgMap {
		pkgs = append(pkgs, p)
	}
	return pkgs
}

func joinExprListWithOrExpr(list []constraint.Expr) constraint.Expr {
	if len(list) == 1 {
		return list[0]
	}
	return &constraint.OrExpr{X: list[0], Y: joinExprListWithOrExpr(list[1:])}
}

func extractBuildDirectives(file *ast.File) (constraint.Expr, error) {
	var plusBuilds []constraint.Expr
	for _, group := range file.Comments {
		if group.Pos() >= file.Package {
			break
		}
		for _, c := range group.List {
			if constraint.IsGoBuild(c.Text) {
				return constraint.Parse(c.Text)
			}
			if !constraint.IsPlusBuild(c.Text) {
				continue
			}
			expr, err := constraint.Parse(c.Text)
			if err != nil {
				return nil, err
			}
			plusBuilds = append(plusBuilds, expr)
		}
	}
	if len(plusBuilds) == 0 {
		return nil, nil
	}
	return joinExprListWithOrExpr(plusBuilds), nil
}

var debugFilterdPackage bool = true

// readDir reads all files in a directory as a Package.
func readDir(path string, testPackage bool, tags map[string]bool) (*Package, error) {
	fset := token.NewFileSet()
	pkgMap, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var filtered bool
	// remove packages which have no files to be built.
	for pname, pkg := range pkgMap {
		// filter pkg.Files by build tags
		for fname, file := range pkg.Files {
			expr, err := extractBuildDirectives(file)
			if err != nil {
				return nil, err
			}
			if expr == nil {
				continue
			}
			if !expr.Eval(func(tag string) bool { return tags[tag] }) {
				delete(pkg.Files, fname)
			}
		}
		if len(pkg.Files) == 0 {
			delete(pkgMap, pname)
			filtered = true
		}
	}
	if len(pkgMap) == 0 {
		if debugFilterdPackage && filtered {
			log.Printf("package:%s is empty because filtered\n", path)
		}
		return &Package{}, nil
	}
	if len(pkgMap) > 2 {
		return nil, fmt.Errorf("multiple packages in directory %s", path)
	}
	pkgs := toPackages(pkgMap)
	// check pkgs includes only target and test packages.
	pkg := pkgs[0]
	if len(pkgs) == 2 {
		testPkg := pkgs[1]
		if len(pkg.Name) > len(testPkg.Name) {
			pkg, testPkg = testPkg, pkg
		}
		if pkg.Name+"_test" != testPkg.Name {
			return nil, fmt.Errorf("multiple non-test packages in directory %s: %s, %s", path, pkg.Name, testPkg.Name)
		}
		// use test package.
		if testPackage {
			pkg = testPkg
		}
	}
	p := &Parser{}
	for _, n := range sortFileNames(pkg.Files) {
		file := pkg.Files[n]
		err := p.ScanFile(file)
		if err != nil {
			return nil, err
		}
	}
	return p.Package, nil
}

func getTags() map[string]bool {
	tagMap := map[string]bool{}
	tagMap[build.Default.GOARCH] = true
	tagMap[build.Default.GOOS] = true
	for _, tags := range [][]string{build.Default.BuildTags, build.Default.ToolTags, build.Default.ReleaseTags} {
		for _, tag := range tags {
			tagMap[tag] = true
		}
	}
	return tagMap
}

// Read reads a file or directory as a Package.
// If you are going to read a directory, see also ReadDir.
func Read(path string) (*Package, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		tags := getTags()
		return readDir(path, false, tags)
	}
	return readFile(path)
}

// ReadDir reads a directory as a Package.  It reads "test" package when
// `testPackage` is set.  It will fail if the directory contains non-test
// multiple packages.
func ReadDir(path string, testPackage bool) (*Package, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %q", path)
	}
	tags := getTags()
	return readDir(path, testPackage, tags)
}
