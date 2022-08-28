package schema

import (
	"fmt"
	"strings"

	"github.com/progrium/macschema/declparse"
)

func PullSchema(l Lookup) Schema {
	t, err := ReadTopic(l)
	fatal(err)

	var s Schema
	s.PullDate = t.LastFetch
	s.Version = Version

	switch t.Type {
	case "Class":
		schemaForClass(&s, t)
	case "Type Alias":
		schemaForTypeAlias(&s, t)
	case "Structure":
		schemaForStruct(&s, t)
	case "Global Variable":
		println(t.Type)
	case "Enumeration":
		schemaForEnum(&s, t)
	case "Function":
		println("TODO")
	default:
		fatal(fmt.Errorf("schema not supported for %s", t.Type))
	}

	return s
}

func identifierFromTopic(t Topic) (id Identifier) {
	id.Declaration = t.Declaration
	id.Name = t.Title
	id.Description = t.Description
	id.Frameworks = t.Frameworks
	id.TopicURL = BaseURL + strings.Replace(t.Path, "/documentation/", "", 1)
	for _, p := range t.Platforms {
		if p == "Deprecated" {
			id.Deprecated = true
		} else {
			id.Platforms = append(id.Platforms, p)
		}
	}
	return
}

func schemaForEnum(s *Schema, t Topic) {
	s.Kind = "enum"

	id := identifierFromTopic(t)

	var en Enum
	if t.Declaration != "" {
		p := declparse.NewStringParser(t.Declaration)
		ast, err := p.Parse()
		if err != nil {
			fatal(fmt.Errorf("%s: %w [%s]", id.TopicURL, err, t.Declaration))
		}
		en = EnumFromAst(*ast.Enum)
	}
	en.Identifier = id

	for _, topic := range t.Topics {
		t, err := ReadTopic(LookupFromPath(topic.Path))
		fatal(err)
		if t.Type != "Enumeration Case" {
			continue
		}
		id := identifierFromTopic(t)
		var ecase Variable
		if t.Declaration != "" {
			p := declparse.NewStringParser(t.Declaration)
			p.Hint = declparse.HintEnumCase
			ast, err := p.Parse()
			if err != nil {
				fatal(fmt.Errorf("%s: %w [%s]", id.TopicURL, err, t.Declaration))
			}
			ecase = VariableFromAst(*ast.Variable)
		}
		ecase.Identifier = id
		en.Cases = append(en.Cases, ecase)
	}

	s.Enum = &en
}

func schemaForStruct(s *Schema, t Topic) {
	s.Kind = "struct"

	id := identifierFromTopic(t)

	var st Struct
	if t.Declaration != "" {
		p := declparse.NewStringParser(t.Declaration)
		ast, err := p.Parse()
		if err != nil {
			fatal(fmt.Errorf("%s: %w [%s]", id.TopicURL, err, t.Declaration))
		}
		st = StructFromAst(*ast.Struct)
	}
	st.Identifier = id

	for _, topic := range t.Topics {
		t, err := ReadTopic(LookupFromPath(topic.Path))
		fatal(err)
		if t.Type != "Instance Property" {
			continue
		}
		id := identifierFromTopic(t)
		var prop Variable
		if t.Declaration != "" {
			p := declparse.NewStringParser(t.Declaration)
			p.Hint = declparse.HintVariable
			ast, err := p.Parse()
			if err != nil {
				fatal(fmt.Errorf("%s: %w [%s]", id.TopicURL, err, t.Declaration))
			}
			prop = VariableFromAst(*ast.Variable)
		}
		prop.Identifier = id
		st.Fields = append(st.Fields, prop)
	}

	s.Struct = &st
}

func schemaForTypeAlias(s *Schema, t Topic) {
	s.Kind = "typealias"

	var ta TypeAlias
	ta.Identifier = identifierFromTopic(t)
	if t.Declaration != "" {
		p := declparse.NewStringParser(t.Declaration)
		ast, err := p.Parse()
		if err != nil {
			fatal(fmt.Errorf("%s: %w [%s]", ta.TopicURL, err, t.Declaration))
		}
		ta.Type = DataTypeFromAst(*ast.TypeAlias)
	}

	for _, topic := range t.Topics {
		t, err := ReadTopic(LookupFromPath(topic.Path))
		fatal(err)
		if t.Type != "Global Variable" {
			continue
		}
		id := identifierFromTopic(t)
		var val Variable
		if t.Declaration != "" {
			p := declparse.NewStringParser(t.Declaration)
			p.Hint = declparse.HintVariable
			ast, err := p.Parse()
			if err != nil {
				fatal(fmt.Errorf("%s: %w [%s]", id.TopicURL, err, t.Declaration))
			}
			val = VariableFromAst(*ast.Variable)
		}
		val.Identifier = id
		if val.Type.Name == ta.Name {
			ta.Values = append(ta.Values, val)
		}
	}

	s.TypeAlias = &ta
}

func schemaForClass(s *Schema, t Topic) {
	s.Kind = "class"

	var c Class
	c.Identifier = identifierFromTopic(t)
	for _, topic := range t.Topics {
		t, err := ReadTopic(LookupFromPath(topic.Path))
		fatal(err)
		if t.Type == "Function" ||
			t.Type == "Enumeration" ||
			t.Type == "Global Variable" ||
			t.Type == "Enumeration Case" ||
			t.Type == "Macro" {
			continue
		}
		var isDeprecated bool
		for _, p := range t.Platforms {
			if p == "Deprecated" {
				isDeprecated = true
			}
		}
		if t.Declaration != "" {
			p := declparse.NewStringParser(t.Declaration)
			ast, err := p.Parse()
			if err != nil {
				fatal(fmt.Errorf("%s: %w [%s]", topic.Path, err, t.Declaration))
			}
			url := BaseURL + strings.Replace(t.Path, "/documentation/", "", 1)
			switch t.Type {
			case "Type Method":
				m := MethodFromAst(*ast.Method)
				m.Description = t.Description
				m.Declaration = t.Declaration
				m.TopicURL = url
				m.Deprecated = isDeprecated
				m.Platforms = t.Platforms
				m.Frameworks = t.Frameworks
				c.TypeMethods = append(c.TypeMethods, m)
			case "Instance Method":
				m := MethodFromAst(*ast.Method)
				m.Description = t.Description
				m.Declaration = t.Declaration
				m.TopicURL = url
				m.Deprecated = isDeprecated
				m.Platforms = t.Platforms
				m.Frameworks = t.Frameworks
				c.InstanceMethods = append(c.InstanceMethods, m)
			case "Type Property":
				p := PropertyFromAst(*ast.Property)
				p.Description = t.Description
				p.Declaration = t.Declaration
				p.TopicURL = url
				p.Deprecated = isDeprecated
				p.Platforms = t.Platforms
				p.Frameworks = t.Frameworks
				c.TypeProperties = append(c.TypeProperties, p)
			case "Instance Property":
				p := PropertyFromAst(*ast.Property)
				p.Description = t.Description
				p.Declaration = t.Declaration
				p.TopicURL = url
				p.Deprecated = isDeprecated
				p.Platforms = t.Platforms
				p.Frameworks = t.Frameworks
				c.InstanceProperties = append(c.InstanceProperties, p)
			default:
			}
		}
	}

	s.Class = &c
}
