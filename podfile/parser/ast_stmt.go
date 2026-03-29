package parser

// Statement is the interface for all build-logic statements.
type Statement interface {
	stmtNode()
	Pos() Position
}

// ArgStmt: arg <args...> [when <condition>]
type ArgStmt struct {
	Args     []Expr
	When     Expr // nil if unconditional
	Position Position
}

// FileStmt: file <path> <content> [<mode>] [when <condition>]
type FileStmt struct {
	Path     Expr
	Content  Expr
	Mode     int // 0 means default (0644)
	When     Expr
	Position Position
}

// MkdirStmt: mkdir <path>
type MkdirStmt struct {
	Path     Expr
	Position Position
}

// AssignStmt: <var> = <expr>
type AssignStmt struct {
	Name     string
	Value    Expr
	Position Position
}

// IfStmt: if <condition> { ... } [else { ... }]
type IfStmt struct {
	Condition Expr
	Body      []Statement
	Else      []Statement // nil if no else
	Position  Position
}

// ForStmt: for <key>, <value> in <expr> { ... }
type ForStmt struct {
	Key      string // loop variable for key (or sole variable)
	Value    string // loop variable for value (empty if single var)
	Iter     Expr   // expression to iterate over
	Body     []Statement
	Position Position
}

func (s *ArgStmt) stmtNode()    {}
func (s *FileStmt) stmtNode()   {}
func (s *MkdirStmt) stmtNode()  {}
func (s *AssignStmt) stmtNode() {}
func (s *IfStmt) stmtNode()     {}
func (s *ForStmt) stmtNode()    {}

func (s *ArgStmt) Pos() Position    { return s.Position }
func (s *FileStmt) Pos() Position   { return s.Position }
func (s *MkdirStmt) Pos() Position  { return s.Position }
func (s *AssignStmt) Pos() Position { return s.Position }
func (s *IfStmt) Pos() Position     { return s.Position }
func (s *ForStmt) Pos() Position    { return s.Position }
