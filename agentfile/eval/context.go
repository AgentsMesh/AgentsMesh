// Package eval implements the Runner-mode AgentFile execution engine.
// It walks the full AST (declarations + build logic) and produces
// a BuildResult — the complete Pod creation instruction.
package eval

// BuildResult is the complete output of evaluating an AgentFile.
// It contains everything needed to create and start a Pod,
// equivalent to CreatePodCommand.
type BuildResult struct {
	// From AGENT declaration
	LaunchCommand string
	// From EXECUTABLE declaration
	Executable string

	// From arg statements
	LaunchArgs []string
	// From ENV declarations + env statements
	EnvVars map[string]string
	// From file statements
	FilesToCreate []FileEntry
	// From mkdir statements
	Dirs []string
	// From PROMPT declaration
	Prompt string // prompt content
	// From PROMPT_POSITION declaration
	PromptPosition string // "prepend", "append", "none"

	// From REPO/BRANCH/GIT_CREDENTIAL declarations
	Sandbox SandboxResult
	// From SETUP declaration
	Setup SetupResult
	// From MCP declaration
	MCPEnabled bool
	// From SKILLS declaration
	Skills []string

	// From REMOVE declarations and remove statements
	RemoveArgs   []string // arg values to remove from LaunchArgs
	RemoveFiles  []string // file paths to remove from FilesToCreate
	RemoveEnvs   []string // env names to remove from EnvVars
	RemoveSkills []string // skill slugs to remove from Skills

	// From MODE declaration
	Mode string // "pty" or "acp"
	// From MODE <name> <args...> declarations (per-mode launch args)
	ModeArgs map[string][]string
	// From CREDENTIAL declaration
	CredentialProfile string // profile name or "runner_host"
}

// FileEntry represents a file to create in the sandbox.
type FileEntry struct {
	Path    string
	Content string
	Mode    int // 0 means default (0644)
}

// SandboxResult holds workspace configuration from declarations.
type SandboxResult struct {
	RepoURL        string // from REPO
	Branch         string // from BRANCH
	CredentialType string // from GIT_CREDENTIAL
}

// SetupResult holds preparation script configuration.
type SetupResult struct {
	Script  string
	Timeout int
}

// Context holds the runtime state during AgentFile evaluation.
type Context struct {
	// Variables is the mutable variable scope.
	Variables map[string]interface{}

	// Credentials maps ENV names to their secret/text values.
	Credentials map[string]string

	// IsRunnerHost: use Runner's local credentials, don't inject.
	IsRunnerHost bool

	// Result accumulates the complete Pod creation instruction.
	Result *BuildResult

	// Builtins maps function names to implementations.
	Builtins map[string]BuiltinFunc
}

// BuiltinFunc is a function callable from AgentFile build logic.
type BuiltinFunc func(args ...interface{}) (interface{}, error)

// NewContext creates a Context with platform-injected values.
func NewContext(vars map[string]interface{}) *Context {
	if vars == nil {
		vars = make(map[string]interface{})
	}
	ctx := &Context{
		Variables: vars,
		Result: &BuildResult{
			EnvVars: make(map[string]string),
		},
		Builtins: make(map[string]BuiltinFunc),
	}
	RegisterBuiltins(ctx)
	return ctx
}

// Get retrieves a variable by name.
func (c *Context) Get(name string) (interface{}, bool) {
	val, ok := c.Variables[name]
	return val, ok
}

// Set assigns a variable value.
func (c *Context) Set(name string, val interface{}) {
	c.Variables[name] = val
}

// GetNested retrieves a value from a nested map.
func GetNested(obj interface{}, key string) (interface{}, bool) {
	if m, ok := obj.(map[string]interface{}); ok {
		val, found := m[key]
		return val, found
	}
	return nil, false
}
