//go:build windows

package fsutil

import (
	"log/slog"
	"os"
	"time"
)

const (
	removeMaxRetries = 3
	removeRetryDelay = 500 * time.Millisecond
)

// RemoveAll removes path and any children it contains.
// On Windows, files may be temporarily locked by antivirus, indexing services,
// or processes that haven't fully released handles. This wrapper retries up to
// 3 times with a 500ms delay between attempts.
func RemoveAll(path string) error {
	var err error
	for i := 0; i < removeMaxRetries; i++ {
		err = os.RemoveAll(path)
		if err == nil {
			return nil
		}
		if i < removeMaxRetries-1 {
			slog.Debug("RemoveAll failed, retrying",
				"path", path,
				"attempt", i+1,
				"error", err)
			time.Sleep(removeRetryDelay)
		}
	}
	return err
}
