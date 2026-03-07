// Package fsutil provides filesystem utilities with cross-platform robustness.
// On Windows, file operations can fail due to file locking (e.g., antivirus scans,
// indexing services, or processes still holding handles). This package provides
// retry-aware wrappers that handle transient failures gracefully.
package fsutil
