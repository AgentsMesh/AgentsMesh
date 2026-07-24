//go:build !(aix || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris)

package mockagent

import "os"

func watchTerminalResize() (<-chan os.Signal, func()) {
	return nil, func() {}
}
