package declparse

import (
	"fmt"

	"github.com/progrium/macschema/pkg/declparse/keywords"
	"github.com/progrium/macschema/pkg/lexer"
)

func parseProperty(p *Parser) (next stateFn, node Node, err error) {
	decl := &PropertyDecl{Attrs: make(map[PropAttr]string)}

	if err := p.expectToken(keywords.PROPERTY); err != nil {
		return nil, nil, err
	}

	if tok, _, _ := p.tb.Scan(); tok == lexer.LPAREN {
		for {
			lit, err := p.expectIdent()
			if err != nil {
				return nil, nil, err
			}

			for _, attr := range PropAttrs() {
				if lit == attr.String() {
					switch attr {
					case PropAttrGetter, PropAttrSetter:
						if err := p.expectToken(lexer.EQ); err != nil {
							return nil, nil, err
						}
						val, err := p.expectIdent()
						if err != nil {
							return nil, nil, err
						}
						decl.Attrs[attr] = val
					default:
						decl.Attrs[attr] = ""
					}
				}
			}

			tok, _, lit := p.tb.Scan()
			if tok == lexer.RPAREN {
				break
			}
			if tok != lexer.COMMA {
				return nil, nil, fmt.Errorf("found %q, expected , or )", lit)
			}
		}
	} else {
		p.tb.Unscan()
	}

	typ, err := p.expectType(false)
	if err != nil {
		return nil, nil, err
	}
	decl.Type = *typ

	decl.Name, err = p.expectIdent()
	if err != nil {
		return nil, nil, err
	}

	return nil, decl, nil
}
