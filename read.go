package srcdom

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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

// readDir reads all files in a directory as a Package.
func readDir(path string, testPackage bool) (*Package, error) {
	fset := token.NewFileSet()
	pkgMap, err := parser.ParseDir(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	if len(pkgMap) == 0 {
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

// Read reads a file or directory as a Package.
func Read(path string) (*Package, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return readDir(path, false)
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
	return readDir(path, testPackage)
}
