package declparser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

type Statement struct {
	Method    *MethodDecl
	Property  *PropertyDecl
	Interface *InterfaceDecl
	Protocol  *ProtocolDecl
}

func (s Statement) String() string {
	if s.Method != nil {
		return s.Method.String()
	}
	if s.Property != nil {
		return s.Property.String()
	}

	if s.Interface != nil {
		return s.Interface.String()
	}
	return ""
}

type ProtocolDecl struct {
	Name      string
	SuperName string
}

func (i ProtocolDecl) String() string {
	b := &strings.Builder{}
	_, _ = fmt.Fprintf(b, "@protocol %s", i.Name)
	if i.SuperName != "" {
		_, _ = fmt.Fprintf(b, " : %s", i.SuperName)
	}
	return b.String()
}

type InterfaceDecl struct {
	Name      string
	SuperName string
}

func (i InterfaceDecl) String() string {
	b := &strings.Builder{}
	_, _ = fmt.Fprintf(b, "@interface %s", i.Name)
	if i.SuperName != "" {
		_, _ = fmt.Fprintf(b, " : %s", i.SuperName)
	}
	return b.String()
}

type PropertyDecl struct {
	Name string
	Type TypeInfo

	// Attributes
	Class     bool
	Readonly  bool
	Weak      bool
	Nonatomic bool
	Copy      bool
	Nullable  bool
	Nonnull   bool
	Retain    bool
	Getter    string
	Setter    string
}

func (p PropertyDecl) String() string {
	b := &strings.Builder{}
	b.WriteString("@property")
	var options []string
	if p.Setter != "" {
		options = append(options, fmt.Sprintf("setter=%s", p.Setter))
	}
	if p.Getter != "" {
		options = append(options, fmt.Sprintf("getter=%s", p.Getter))
	}
	if p.Class {
		options = append(options, "class")
	}
	if p.Readonly {
		options = append(options, "readonly")
	}
	if p.Copy {
		options = append(options, "copy")
	}
	if p.Nonatomic {
		options = append(options, "nonatomic")
	}
	if p.Weak {
		options = append(options, "weak")
	}
	if p.Retain {
		options = append(options, "retain")
	}
	if len(options) != 0 {
		b.WriteString("(")
		b.WriteString(strings.Join(options, ", "))
		b.WriteString(")")
	}

	b.WriteString(" ")
	b.WriteString(p.Type.Name)
	b.WriteString(" ")
	if p.Type.IsPtr {
		b.WriteString("*")
	}
	b.WriteString(p.Name)
	b.WriteString(";")
	return b.String()
}

type FunctionDecl struct {
	Name       string
	ReturnType TypeInfo
	Args       []ArgInfo
	IsBlock    bool
}

func (f FunctionDecl) String() string {
	b := &strings.Builder{}
	b.WriteString(f.ReturnType.String())
	if f.IsBlock {
		b.WriteString("(^")
		b.WriteString(f.Name)
		b.WriteString(")")
	} else {
		b.WriteString(f.Name)
	}
	b.WriteString("(")
	for i, arg := range f.Args {
		b.WriteString(arg.Type.String())
		b.WriteString(" ")
		b.WriteString(arg.Name)
		if i < len(f.Args)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString(")")
	return b.String()
}

type MethodDecl struct {
	TypeMethod bool // instance method otherwise
	ReturnType TypeInfo
	NameParts  []string
	Args       []ArgInfo
}

func (m *MethodDecl) Name() string {
	if len(m.NameParts) == 0 {
		return ""
	}
	if len(m.NameParts) == 1 {
		return m.NameParts[0]
	}
	return strings.Join(append(m.NameParts, ""), ":")
}

func (m MethodDecl) String() string {
	b := &strings.Builder{}
	if m.TypeMethod {
		b.WriteString("+")
	} else {
		b.WriteString("-")
	}
	b.WriteString(" ")
	b.WriteString(fmt.Sprintf("(%s)", m.ReturnType.String()))
	b.WriteString(m.NameParts[0])
	for i, arg := range m.Args {
		if i != 0 {
			b.WriteString(" \n")
			b.WriteString(m.NameParts[i])
		}
		b.WriteString(":")
		b.WriteString(arg.String())

	}
	b.WriteString(";")
	return b.String()
}

type TypeInfo struct {
	Name     string
	IsPtr    bool
	IsConst  bool
	IsKindOf bool
	Block    *FunctionDecl
	Params   []TypeInfo
}

func (t TypeInfo) String() string {
	if t.Block != nil {
		return t.Block.String()
	}
	b := &strings.Builder{}
	b.WriteString(t.Name)
	if len(t.Params) > 0 {
		b.WriteString("<")
		for _, param := range t.Params {
			b.WriteString(param.String())
		}
		b.WriteString(">")
	}
	if t.IsPtr {
		b.WriteString(" *")
	}
	return b.String()
}

type ArgInfo struct {
	Name string
	Type TypeInfo
}

func (arg ArgInfo) String() string {
	b := &strings.Builder{}
	b.WriteString(fmt.Sprintf("(%s)", arg.Type.String()))
	b.WriteString(arg.Name)
	return b.String()
}

type Parser struct {
	s   *scanner
	buf struct {
		tok token  // last read token
		lit string // last read literal
		n   int    // buffer size (max=1)
	}
}

func NewParser(r io.Reader) *Parser {
	return &Parser{s: &scanner{r: bufio.NewReader(r)}}
}

func NewStringParser(s string) *Parser {
	return &Parser{s: &scanner{r: bufio.NewReader(bytes.NewBufferString(s))}}
}

// scan returns the next token from the underlying scanner.
// If a token has been unscanned then read that instead.
func (p *Parser) scan() (tok token, lit string) {
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}

	tok, lit = p.s.Scan()

	// ignore whitespace
	if tok == WS {
		tok, lit = p.s.Scan()
	}

	p.buf.tok, p.buf.lit = tok, lit

	return
}

func (p *Parser) unscan() { p.buf.n = 1 }

func (p *Parser) parseInterface() (*InterfaceDecl, error) {
	decl := &InterfaceDecl{}

	tok, lit := p.scan()
	if tok != IDENT {
		return nil, fmt.Errorf("found %q, expected identifier", lit)
	}
	decl.Name = lit

	tok, lit = p.scan()
	if tok == COLON {
		tok, lit = p.scan()
		if tok != IDENT {
			return nil, fmt.Errorf("found %q, expected identifier", lit)
		}
		decl.SuperName = lit
	} else {
		p.unscan()
	}

	return decl, nil
}

func (p *Parser) parseProtocol() (*ProtocolDecl, error) {
	decl := &ProtocolDecl{}

	tok, lit := p.scan()
	if tok != IDENT {
		return nil, fmt.Errorf("found %q, expected identifier", lit)
	}
	decl.Name = lit

	tok, lit = p.scan()
	if tok == COLON {
		tok, lit = p.scan()
		if tok != IDENT {
			return nil, fmt.Errorf("found %q, expected identifier", lit)
		}
		decl.SuperName = lit
	} else {
		p.unscan()
	}

	return decl, nil
}

func (p *Parser) Parse() (*Statement, error) {
	tok, lit := p.scan()
	switch tok {
	// TODO: typedef, var? ... const? [can be function apparently]
	case PLUS, MINUS:
		p.unscan()
		decl, err := p.parseMethod()
		return &Statement{Method: decl}, err
	case PROPERTY:
		decl, err := p.parseProperty()
		return &Statement{Property: decl}, err
	case INTERFACE:
		decl, err := p.parseInterface()
		return &Statement{Interface: decl}, err
	case PROTOCOL:
		decl, err := p.parseProtocol()
		return &Statement{Protocol: decl}, err
	default:
		// TODO: parseFunction
		return nil, fmt.Errorf("found %q, expected method (+,-) or keyword (@...)", lit)
	}
}
