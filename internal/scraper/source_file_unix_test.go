//go:build unix

package scraper

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"jargonary/internal/config"
	"jargonary/internal/store"
	"jargonary/internal/vocabulary"
)

func TestFIFORejectionAndNonblockingOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "terms.yaml")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	// Release a blocked reader on failure so a regression does not hang tests.
	t.Cleanup(func() {
		fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_NONBLOCK, 0600)
		if err == nil {
			syscall.Close(fd)
		}
	})
	t.Run("open resists replacement with FIFO", func(t *testing.T) {
		done := make(chan error, 1)
		go func() {
			f, err := openSourceFile(path)
			if err == nil {
				err = f.Close()
			}
			done <- err
		}()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("opening a FIFO blocked before descriptor validation")
		}
	})
	t.Run("scrape retains snapshot and worker shuts down", func(t *testing.T) {
		st, err := store.Open(filepath.Join(t.TempDir(), "store.db"))
		if err != nil {
			t.Fatal(err)
		}
		defer st.Close()
		ctx := context.Background()
		d := vocabulary.Dictionary{Version: "1", Name: "Last good", Entries: []vocabulary.Entry{{ID: "term", Term: "Term", Senses: []vocabulary.Sense{{ID: "one", Definition: "Retained"}}}}}
		if _, err := st.Replace(ctx, "local", d, "", ""); err != nil {
			t.Fatal(err)
		}
		c := config.Config{Timeout: config.Duration(time.Second), MaxBytes: 4096, Sources: []config.Source{{ID: "local", File: path, Format: "yaml", ScrapeInterval: config.Duration(time.Minute)}}}
		s := New(c, st)
		done := make(chan store.Status, 1)
		go func() { done <- s.Once(ctx, c.Sources[0], store.Status{}) }()
		select {
		case status := <-done:
			if status.Failures != 1 || status.Running {
				t.Fatalf("FIFO not rejected: %+v", status)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("FIFO scrape exceeded timeout")
		}
		snapshots, err := st.Snapshots(ctx)
		if err != nil || len(snapshots) != 1 || len(snapshots[0].Dictionary.Entries) != 1 {
			t.Fatalf("FIFO lost last good snapshot: %+v %v", snapshots, err)
		}
		runCtx, cancel := context.WithCancel(ctx)
		runDone := make(chan error, 1)
		go func() { runDone <- s.Run(runCtx) }()
		cancel()
		select {
		case <-runDone:
		case <-time.After(2 * time.Second):
			t.Fatal("worker did not stop after cancellation")
		}
	})
}

func TestRegularSourceFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "terms.json")
	body := `{"version":"1","name":"Local","entries":[]}`
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{MaxBytes: 4096}, nil)
	got, _, _, _, err := s.fetch(context.Background(), config.Source{File: path}, nil)
	if err != nil || string(got) != body {
		t.Fatalf("regular source: %q %v", got, err)
	}
}
