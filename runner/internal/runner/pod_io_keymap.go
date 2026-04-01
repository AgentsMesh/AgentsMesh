package runner

// ptyKeyMap maps human-readable key names to terminal escape sequences.
var ptyKeyMap = map[string]string{
	"enter":     "\r",
	"escape":    "\x1b",
	"tab":       "\t",
	"backspace": "\x7f",
	"delete":    "\x1b[3~",
	"ctrl+c":    "\x03",
	"ctrl+d":    "\x04",
	"ctrl+u":    "\x15",
	"ctrl+l":    "\x0c",
	"ctrl+z":    "\x1a",
	"ctrl+a":    "\x01",
	"ctrl+e":    "\x05",
	"ctrl+k":    "\x0b",
	"ctrl+w":    "\x17",
	"up":        "\x1b[A",
	"down":      "\x1b[B",
	"right":     "\x1b[C",
	"left":      "\x1b[D",
	"home":      "\x1b[H",
	"end":       "\x1b[F",
	"pageup":    "\x1b[5~",
	"pagedown":  "\x1b[6~",
	"shift+tab": "\x1b[Z",
}
