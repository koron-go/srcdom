package srcdom_test

import (
	"io/fs"
	"path/filepath"
	"runtime"
	"testing"

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
