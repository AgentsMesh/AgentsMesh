//go:build windows

package mcp

// TryReclaimPort is a no-op on Windows.
// Windows uses service-manager-based restart which handles port cleanup.
func TryReclaimPort(port int) bool {
	return false
}
