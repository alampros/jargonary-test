//go:build unix

package config

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestDiscoverySkipsFIFO(t *testing.T) {
	dir := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(dir, "pipe.yaml"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "regular.json"), []byte(`{"version":"1","name":"Local","entries":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(writeConfig(t, "terms_dir: "+dir+"\n"))
	if err != nil || len(c.Sources) != 1 || c.Sources[0].ID != "regular" {
		t.Fatalf("discovery should select only regular files: %+v %v", c.Sources, err)
	}
}
