//go:build unix

package scraper

import (
	"os"
	"syscall"
)

// O_NONBLOCK prevents a file replaced with a FIFO from blocking before f.Stat.
// It has no effect on regular file reads.
func openSourceFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
