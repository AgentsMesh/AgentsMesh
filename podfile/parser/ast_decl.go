package parser

// Declaration is the interface for all upper-case directives.
type Declaration interface {
	declNode()
	Pos() Position
}

// AgentDecl: AGENT <command>
type AgentDecl struct {
	Command  string
	Position Position
}

// ExecutableDecl: EXECUTABLE <name>
type ExecutableDecl struct {
	Name     string
	Position Position
}

// ConfigDecl: CONFIG <name> <type_expr> [= <default>]
type ConfigDecl struct {
	Name     string
	TypeName string      // "boolean", "string", "number", "secret", "select"
	Options  []string    // SELECT options (empty for other types)
	Default  interface{} // default value (string, bool, float64, or nil)
	Position Position
}

// EnvDecl: ENV <name> SECRET|TEXT [OPTIONAL]  or  ENV <name> = <value>
type EnvDecl struct {
	Name     string
	Source   string // "secret" or "text" (credential), empty for fixed value
	Value    string // fixed value (when Source is empty)
	Optional bool
	Position Position
}

// RepoDecl: REPO <expr>
type RepoDecl struct {
	Value    Expr
	Position Position
}

// BranchDecl: BRANCH <expr>
type BranchDecl struct {
	Value    Expr
	Position Position
}

// GitCredentialDecl: GIT_CREDENTIAL <type>
type GitCredentialDecl struct {
	Type     string
	Position Position
}

// McpDecl: MCP ON|OFF
type McpDecl struct {
	Enabled  bool
	Position Position
}

// SkillsDecl: SKILLS <slug1>, <slug2>, ...
type SkillsDecl struct {
	Slugs    []string
	Position Position
}

// SetupDecl: SETUP [timeout=<n>] <<HEREDOC
type SetupDecl struct {
	Script   string
	Timeout  int
	Position Position
}

// RemoveDecl: REMOVE ENV <name> | REMOVE SKILLS <slug>
type RemoveDecl struct {
	Target   string // "ENV", "SKILLS", "CONFIG"
	Name     string // the specific item to remove
	Position Position
}

// ModeDecl: MODE pty | MODE acp
type ModeDecl struct {
	Mode     string // "pty" or "acp"
	Position Position
}

// CredentialDecl: CREDENTIAL "profile-name" | CREDENTIAL runner_host
type CredentialDecl struct {
	ProfileName string // profile name or "runner_host"
	Position    Position
}

// PromptDecl: PROMPT "initial prompt content"
type PromptDecl struct {
	Content  string
	Position Position
}

// PromptPositionDecl: PROMPT_POSITION prepend | append | none
type PromptPositionDecl struct {
	Mode     string // "prepend", "append", "none"
	Position Position
}

func (d *AgentDecl) declNode()          {}
func (d *ExecutableDecl) declNode()     {}
func (d *ConfigDecl) declNode()         {}
func (d *EnvDecl) declNode()            {}
func (d *RepoDecl) declNode()           {}
func (d *BranchDecl) declNode()         {}
func (d *GitCredentialDecl) declNode()  {}
func (d *McpDecl) declNode()            {}
func (d *SkillsDecl) declNode()         {}
func (d *SetupDecl) declNode()          {}
func (d *RemoveDecl) declNode()         {}
func (d *ModeDecl) declNode()           {}
func (d *CredentialDecl) declNode()     {}
func (d *PromptDecl) declNode()         {}
func (d *PromptPositionDecl) declNode() {}

func (d *AgentDecl) Pos() Position          { return d.Position }
func (d *ExecutableDecl) Pos() Position     { return d.Position }
func (d *ConfigDecl) Pos() Position         { return d.Position }
func (d *EnvDecl) Pos() Position            { return d.Position }
func (d *RepoDecl) Pos() Position           { return d.Position }
func (d *BranchDecl) Pos() Position         { return d.Position }
func (d *GitCredentialDecl) Pos() Position  { return d.Position }
func (d *McpDecl) Pos() Position            { return d.Position }
func (d *SkillsDecl) Pos() Position         { return d.Position }
func (d *SetupDecl) Pos() Position          { return d.Position }
func (d *RemoveDecl) Pos() Position         { return d.Position }
func (d *ModeDecl) Pos() Position           { return d.Position }
func (d *CredentialDecl) Pos() Position     { return d.Position }
func (d *PromptDecl) Pos() Position         { return d.Position }
func (d *PromptPositionDecl) Pos() Position { return d.Position }
