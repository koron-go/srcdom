package srcdom_test

import (
	"io/fs"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/koron-go/srcdom"
)

func TestReadDir(t *testing.T) {
	p, err := srcdom.ReadDir(".", false)
	if err != nil {
		t.Fatalf("failed to read target package: %s", err)
	}
	if p.Name != "srcdom" {
		t.Errorf("mimatch name of target package: want=%s got=%s", "srcdorm", p.Name)
	}

	p2, err := srcdom.ReadDir(".", true)
	if err != nil {
		t.Fatalf("failed to read test package: %s", err)
	}
	if p2.Name != "srcdom_test" {
		t.Errorf("mimatch name of target package: want=%s got=%s", "srcdorm_test", p.Name)
	}
}

func TestReadGOROOT(t *testing.T) {
	srcdir := filepath.Join(runtime.GOROOT(), "src")
	err := filepath.WalkDir(srcdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		switch d.Name() {
		case "internal", "testdata", "vendor":
			return fs.SkipDir
		}
		_, err = srcdom.ReadDir(path, false)
		if err != nil {
			t.Errorf("failed to read pacakge: %s", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadFile(t *testing.T) {
	got, err := srcdom.Read(filepath.Join("_testdata", "test1.go"))
	if err != nil {
		t.Fatal(err)
	}
	want := srcdom.Package{
		Name: "testdata",
		Funcs: []*srcdom.Func{
			{
				Name:    "Func1",
				Params:  []*srcdom.Var{{Name: "arg1", Type: "string"}},
				Results: []*srcdom.Var{{Type: "error"}},
			},
			{Name: "Func0"},
		},
		Values: []*srcdom.Value{
			{Name: "VarFoo", Type: "int"},
			{Name: "VarBar", Type: "string"},
			{Name: "varPriv", Type: "float64"},
		},
	}
	if d := cmp.Diff(&want, got, cmpopts.IgnoreUnexported(srcdom.Package{})); d != "" {
		t.Errorf("unmatch srcdom.Package: -want +got\n%s", d)
	}
	pkg := got

	t.Run("Value", func(t *testing.T) {
		for _, c := range []struct {
			name  string
			ispub bool
			want  srcdom.Value
		}{
			{"VarFoo", true, srcdom.Value{Name: "VarFoo", Type: "int"}},
			{"VarBar", true, srcdom.Value{Name: "VarBar", Type: "string"}},
			{"varPriv", false, srcdom.Value{Name: "varPriv", Type: "float64"}},
		} {
			got, ok := pkg.Value(c.name)
			if !ok {
				t.Errorf("value:%s not found", c.name)
				continue
			}
			if d := cmp.Diff(&c.want, got); d != "" {
				t.Errorf("value:%s unmatch: -want +got\n%s", c.name, d)
			}
			gotPub := got.IsPublic()
			if gotPub != c.ispub {
				t.Errorf("value:%s IsPublic unmatch: want=%t got=%t", c.name, c.ispub, gotPub)
			}
		}
	})

	t.Run("ValueNames", func(t *testing.T) {
		got := pkg.ValueNames()
		want := []string{
			"VarBar",
			"VarFoo",
			"varPriv",
		}
		if d := cmp.Diff(want, got); d != "" {
			t.Errorf("unmatch ValueNames() result: -want +got\n%s", d)
		}
	})

	t.Run("Func", func(t *testing.T) {
		for _, c := range []struct {
			name  string
			ispub bool
			want  srcdom.Func
		}{
			{"Func1", true, srcdom.Func{
				Name:    "Func1",
				Params:  []*srcdom.Var{{Name: "arg1", Type: "string"}},
				Results: []*srcdom.Var{{Type: "error"}},
			}},
			{"Func0", true, srcdom.Func{Name: "Func0"}},
		} {
			got, ok := pkg.Func(c.name)
			if !ok {
				t.Errorf("func:%s not found", c.name)
				continue
			}
			if d := cmp.Diff(&c.want, got); d != "" {
				t.Errorf("func:%s unmatch: -want +got\n%s", c.name, d)
			}
			gotPub := got.IsPublic()
			if gotPub != c.ispub {
				t.Errorf("func:%s IsPublic unmatch: want=%t got=%t", c.name, c.ispub, gotPub)
			}
		}
	})

	t.Run("FuncNames", func(t *testing.T) {
		got := pkg.FuncNames()
		want := []string{
			"Func0",
			"Func1",
		}
		if d := cmp.Diff(want, got); d != "" {
			t.Errorf("unmatch FuncNames() result: -want +got\n%s", d)
		}
	})
}
