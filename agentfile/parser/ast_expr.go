package parser

// Expr is the interface for all expressions.
type Expr interface {
	exprNode()
}

// StringLit: "hello"
type StringLit struct{ Value string }

// NumberLit: 42, 3.14, 0600
type NumberLit struct{ Value string } // kept as string to preserve octal

// BoolLit: true, false
type BoolLit struct{ Value bool }

// Ident: variable name
type Ident struct{ Name string }

// DotExpr: config.model, sandbox.root
type DotExpr struct {
	Left  Expr
	Field string
}

// BinaryExpr: a + b, a == b, a != b, a and b, a or b
type BinaryExpr struct {
	Left  Expr
	Op    string // "+", "==", "!=", "and", "or"
	Right Expr
}

// UnaryExpr: not x
type UnaryExpr struct {
	Op      string // "not"
	Operand Expr
}

// CallExpr: json_merge(a, b), json({...})
type CallExpr struct {
	Func string
	Args []Expr
}

// ObjectLit: { key: value, key2: value2 }
type ObjectLit struct {
	Fields []ObjectField
}

// ObjectField is a key-value pair in an object literal.
type ObjectField struct {
	Key   string
	Value Expr
}

// HeredocLit: <<EOF ... EOF
type HeredocLit struct{ Content string }

// ListLit: ["a", "b", 42]
type ListLit struct{ Elements []Expr }

func (e *StringLit) exprNode()  {}
func (e *NumberLit) exprNode()  {}
func (e *BoolLit) exprNode()    {}
func (e *Ident) exprNode()      {}
func (e *DotExpr) exprNode()    {}
func (e *BinaryExpr) exprNode() {}
func (e *UnaryExpr) exprNode()  {}
func (e *CallExpr) exprNode()   {}
func (e *ObjectLit) exprNode()  {}
func (e *HeredocLit) exprNode() {}
func (e *ListLit) exprNode()    {}
