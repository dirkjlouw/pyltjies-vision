package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(path, []byte("# key\nPV_A=\"one\"\nexport PV_B=two\nPV_C=file\nnonsense\n"), 0o600)
	t.Setenv("PV_C", "shell")
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Unsetenv("PV_A"); os.Unsetenv("PV_B") })
	if os.Getenv("PV_A") != "one" || os.Getenv("PV_B") != "two" || os.Getenv("PV_C") != "shell" {
		t.Fatal(os.Getenv("PV_A"), os.Getenv("PV_B"), os.Getenv("PV_C"))
	}
	if err := Load(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatal("a missing file is fine")
	}
}
