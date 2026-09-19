package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# comment\nADMIN_USERNAME=fromfile\nADMIN_PASSWORD=secret\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ADMIN_USERNAME", "already")
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("ADMIN_USERNAME") != "already" {
		t.Fatalf("không ghi đè env có sẵn")
	}
	if os.Getenv("ADMIN_PASSWORD") != "secret" {
		t.Fatalf("ADMIN_PASSWORD=%q", os.Getenv("ADMIN_PASSWORD"))
	}
}
