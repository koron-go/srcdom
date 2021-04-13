package srcdom_test

import (
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
