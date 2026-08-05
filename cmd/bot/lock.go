package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// acquireInstanceLock prevents a second bot process (local go run vs systemd) from polling.
// Lock file stays open for the process lifetime.
func acquireInstanceLock() (*os.File, error) {
	path := os.Getenv("BOT_LOCK_PATH")
	if path == "" {
		path = filepath.Join(".", "bot.lock")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("another bot instance holds %s: %w", path, err)
	}
	return f, nil
}
