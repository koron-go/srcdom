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

// readDir reads all files in a directory as a Package.
func readDir(path string) (*Package, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return &Package{}, nil
	}
	if len(pkgs) > 1 {
		return nil, fmt.Errorf("multiple packages in directory %s", path)
	}
	p := &Parser{}
	for _, pkg := range pkgs {
		for _, n := range sortFileNames(pkg.Files) {
			file := pkg.Files[n]
			err := p.ScanFile(file)
			if err != nil {
				return nil, err
			}
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
		return readDir(path)
	}
	return readFile(path)
}
