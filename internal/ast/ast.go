package ast

type NodeType string

const (
	Program       NodeType = "Program"
	VarDecl       NodeType = "VarDecl"
	FnDecl        NodeType = "FnDecl"
	ClassDecl     NodeType = "ClassDecl"
	EnumDecl      NodeType = "EnumDecl"
	NamespaceDecl NodeType = "NamespaceDecl"
	ComponentDecl NodeType = "ComponentDecl"
	ImportDecl    NodeType = "ImportDecl"
	ExportDecl    NodeType = "ExportDecl"
	LunexRequire  NodeType = "LunexRequire"
	UseStmt       NodeType = "UseStmt"
	ImmutableDecl NodeType = "ImmutableDecl"
	UsingDecl     NodeType = "UsingDecl"
	Block         NodeType = "Block"
	ExprStmt      NodeType = "ExprStmt"
	LogStmt       NodeType = "LogStmt"
	ReturnStmt    NodeType = "ReturnStmt"
	ThrowStmt     NodeType = "ThrowStmt"
	RaiseStmt     NodeType = "RaiseStmt"
	BreakStmt     NodeType = "BreakStmt"
	ContinueStmt  NodeType = "ContinueStmt"
	IfStmt        NodeType = "IfStmt"
	UnlessStmt    NodeType = "UnlessStmt"
	WhileStmt     NodeType = "WhileStmt"
	ForStmt       NodeType = "ForStmt"
	ForOfStmt     NodeType = "ForOfStmt"
	EachInStmt    NodeType = "EachInStmt"
	RepeatStmt    NodeType = "RepeatStmt"
	LoopStmt      NodeType = "LoopStmt"
	GuardStmt     NodeType = "GuardStmt"
	DeferStmt     NodeType = "DeferStmt"
	MatchStmt     NodeType = "MatchStmt"
	TryStmt       NodeType = "TryStmt"
	SpawnStmt     NodeType = "SpawnStmt"
	SelectStmt    NodeType = "SelectStmt"
	WithStmt      NodeType = "WithStmt"
	AssertStmt    NodeType = "AssertStmt"
	HaveStmt      NodeType = "HaveStmt"
	IfHaveStmt    NodeType = "IfHaveStmt"
	IfSetStmt     NodeType = "IfSetStmt"
	DeleteStmt    NodeType = "DeleteStmt"
	Identifier    NodeType = "Identifier"
	NumberLit     NodeType = "NumberLit"
	StringLit     NodeType = "StringLit"
	TemplateLit   NodeType = "TemplateLit"
	BoolLit       NodeType = "BoolLit"
	NullLit       NodeType = "NullLit"
	UndefinedLit  NodeType = "UndefinedLit"
	ArrayLit      NodeType = "ArrayLit"
	ObjectLit     NodeType = "ObjectLit"
	RegexLit      NodeType = "RegexLit"
	FnExpr        NodeType = "FnExpr"
	ArrowFn       NodeType = "ArrowFn"
	CallExpr      NodeType = "CallExpr"
	NewExpr       NodeType = "NewExpr"
	MemberExpr    NodeType = "MemberExpr"
	BinaryExpr    NodeType = "BinaryExpr"
	UnaryExpr     NodeType = "UnaryExpr"
	AssignExpr    NodeType = "AssignExpr"
	TernaryExpr   NodeType = "TernaryExpr"
	LogicalExpr   NodeType = "LogicalExpr"
	SpreadExpr    NodeType = "SpreadExpr"
	PipelineExpr  NodeType = "PipelineExpr"
	SequenceExpr  NodeType = "SequenceExpr"
	NotExpr       NodeType = "NotExpr"
	HaveExpr      NodeType = "HaveExpr"
	TrySafeExpr   NodeType = "TrySafeExpr"
	RangeExpr     NodeType = "RangeExpr"
	SleepExpr     NodeType = "SleepExpr"
	ChannelExpr   NodeType = "ChannelExpr"
	NaxImportExpr NodeType = "NaxImportExpr"
	AtImportExpr  NodeType = "AtImportExpr"
	StructLit     NodeType = "StructLit"
	ThisExpr      NodeType = "ThisExpr"
	SuperExpr     NodeType = "SuperExpr"
	VoidExpr      NodeType = "VoidExpr"
	TypeofExpr    NodeType = "TypeofExpr"
	DeleteExpr    NodeType = "DeleteExpr"
	SatisfiesExpr NodeType = "SatisfiesExpr"
	DecoratedExpr NodeType = "DecoratedExpr"
)

type Node struct {
	Type NodeType
	Line int
	Col  int

	Name        string
	Value       interface{}
	IsConst     bool
	IsAbstract  bool
	IsStatic    bool
	IsPrivate   bool
	IsPublic    bool
	IsProtected bool
	IsReadonly  bool
	IsOverride  bool
	Optional    bool
	Computed    bool
	Prefix      bool
	Rest        bool
	TypeOnly    bool

	Body       *Node
	Init       *Node
	Test       *Node
	Alternate  *Node
	Consequent *Node
	Left       *Node
	Right      *Node
	Object     *Node
	Callee     *Node
	Arg        *Node
	Expr       *Node
	Stmt       *Node
	Subject    *Node
	Source     string
	Op         string
	Prop       interface{}

	Params        []*Param
	Args          []*Node
	Elements      []*Node
	Body_         []*Node
	Cases         []*MatchCase
	Members       []*EnumMember
	Specifiers    []*ImportSpec
	Decorators    []*Node
	Properties    []*ObjProp
	Parts         interface{}
	Methods       []*ClassMember
	Modules       []string
	Exprs         []*Node
	Count         *Node
	Ms            *Node
	Lo            *Node
	Hi            *Node
	InExpr        interface{}
	Alias         string
	MatchMode     string
	IsGuard       bool
	ID            int
	Pattern       string
	Flags         string
	DefaultImport string
	Namespace     string
	Declaration   *Node
	FieldName     string
	TypeAnn       interface{}
	Destructure   interface{}
	Extends       *Node
	SuperClass    *Node
	URL           string
	BindingName   string
	Channel       *Node
	CatchParam    string
	CatchBlock    *Node
	FinallyBlock  *Node
	Guard         *Node
	Binding       string
	SelectCases   []*SelectCase

	// NumCache holds the parsed float64 for a NumberLit node, computed once
	// and reused on every subsequent evaluation instead of re-parsing the
	// source string representation each time. Set via atomic.Pointer semantics
	// by the interpreter (see runtime.evalNumber); nil means "not yet cached".
	NumCache *float64

	// NeedsScopeCache caches, for a Block node, whether it declares any
	// bindings of its own (var/const/fn) and therefore needs a fresh
	// Environment. Blocks that only contain expressions/assignments (e.g. a
	// while-loop body like `{ i = i + 1 }`) can safely execute directly in
	// the parent's Environment, avoiding an alloc/pool round-trip on every
	// iteration. nil means "not yet computed".
	NeedsScopeCache *bool

	// ScopeInfo is set by internal/resolver on every node that introduces
	// its own lexical frame (function bodies, for/catch/have/match-case
	// frames, and blocks that declare bindings). It records, in binding
	// order, the names local to that frame, so the runtime can back it
	// with a slot slice instead of a map. nil means "not resolved" (either
	// resolution has not run, or this node never introduces its own
	// frame) and the runtime falls back to its existing map-based
	// Environment for it.
	ScopeInfo *ScopeInfo

	// ResolvedAddr is set by internal/resolver on Identifier nodes (and on
	// assignment-target Identifiers) whose binding was proven statically
	// addressable: walk Hops parent Environments, then index Slot. nil
	// means "not resolved" -- either the name is a global/module-level
	// binding, or resolution had to cross a dynamic frame (see
	// ScopeInfo.Dynamic) -- and the runtime falls back to its existing
	// name-based Environment.Get/Set.
	ResolvedAddr *ResolvedAddr
}

// ScopeInfo records the statically-known bindings of one lexical frame.
// See the resolver package for how it is computed.
type ScopeInfo struct {
	Names   []string
	Dynamic bool
}

// ResolvedAddr is the statically-resolved address of an identifier
// reference: walk Hops parent Environments from the current one, then
// index Slot into that frame's slot slice.
type ResolvedAddr struct {
	Hops int
	Slot int
}

type Param struct {
	Name        string
	TypeAnn     interface{}
	DefaultVal  *Node
	Rest        bool
	Optional    bool
	Destructure interface{}

	// ResolvedSlot is set by internal/resolver to this parameter's slot
	// index within its function's frame (always Hops 0 -- a parameter is
	// always bound directly into the frame that declares it, never a
	// parent). -1 means unresolved (destructured params declare their
	// own sub-bindings instead and leave this at -1).
	ResolvedSlot int
}

type MatchCase struct {
	Patterns  []*MatchPattern
	Guard     *Node
	Body      *Node
	IsDefault bool
}

type MatchPattern struct {
	Kind   string
	Value  interface{}
	Name   string
	Items  []*MatchPattern
	Props  []*MatchProp
	Fields []*MatchPattern
	Path   string
}

type MatchProp struct {
	Key   string
	Alias string
}

type EnumMember struct {
	Name string
	Init *Node
}

type ImportSpec struct {
	Imported string
	Local    string
	Exported string
}

type ObjProp struct {
	Kind     string
	Key      interface{}
	Value    *Node
	Computed bool
	Arg      *Node
	Params   []*Param
	Body     *Node
	IsGet    bool
	IsSet    bool

	// ShorthandAddr is set by internal/resolver for Kind=="shorthand"
	// properties (`{ x }`, equivalent to `{ x: x }`), whose implicit
	// variable reference has no Identifier AST node of its own to carry
	// a ResolvedAddr. nil means unresolved; falls back to the existing
	// env.Get(key) lookup.
	ShorthandAddr *ResolvedAddr
}

type ClassMember struct {
	Kind       string
	Name       string
	Params     []*Param
	Body       *Node
	IsStatic   bool
	IsPrivate  bool
	IsAbstract bool
	IsGet      bool
	IsSet      bool
	Init       *Node
	TypeAnn    interface{}
}

type SelectCase struct {
	Binding string
	Channel *Node
	Body    *Node
}
