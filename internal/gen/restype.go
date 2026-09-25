package gen

import (
	"strings"
)

// ---------- schema helpers ----------

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func strVal(m map[string]any, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func boolVal(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

// refName returns the $ref target if the node is / contains a reference.
func refName(node any) (string, bool) {
	m, ok := asMap(node)
	if !ok {
		return "", false
	}
	if r, ok := strVal(m, "$ref"); ok {
		return r, true
	}
	return "", false
}

func (g *generator) isKnownType(name string) bool {
	for _, k := range g.typeOrder {
		if k == name {
			return true
		}
	}
	return false
}

// refGoType maps a $ref target to a Go type name, recording unresolved refs.
func (g *generator) refGoType(ref string) (string, string) {
	name := typeName(ref)
	if g.isKnownType(ref) {
		if k, ok := g.knownKinds[name]; ok {
			return name, k
		}
		return name, "any" // kind not yet computed (forward ref); treat generically
	}
	g.missing[ref] = true
	return name, "string" // stub kind
}

// primitiveGoType maps a JSON-schema primitive name. ok=false if not primitive.
func primitiveGoType(t string) (string, string, bool) {
	switch strings.ToLower(t) {
	case "string":
		return "string", "string", true
	case "integer":
		return "int", "int", true
	case "number":
		return "float64", "float", true
	case "boolean":
		return "bool", "bool", true
	case "any":
		return "any", "any", true
	case "null":
		return "any", "any", true
	case "object":
		return "map[string]any", "map", true
	case "array":
		return "[]any", "slice", true
	}
	return "", "", false
}

// goTypeOf resolves a schema node to a Go type expression.
// If the node is an anonymous object that needs a declaration, hint names it
// and the declaration is appended to g.extraDefs (or returned by callers that
// pass collect=false). kind is one of: struct,slice,string,int,float,bool,any,map,ptr.
func (g *generator) goTypeOf(node any, hint string) (typ, kind string) {
	if node == nil {
		return "any", "any"
	}
	if s, ok := node.(string); ok {
		if p, k, ok := primitiveGoType(s); ok {
			return p, k
		}
		// A bare string naming another type (e.g. "type": "Video.Ratings").
		return g.refGoType(s)
	}
	if arr, ok := node.([]any); ok {
		return g.unionGoType(arr, hint)
	}
	m, ok := asMap(node)
	if !ok {
		return "any", "any"
	}
	if ref, ok := strVal(m, "$ref"); ok {
		return g.refGoType(ref)
	}
	t, hasType := m["type"]
	if !hasType {
		// No "type": either extends-only or properties-only (treat as object).
		if _, ok := m["extends"]; ok {
			return g.objectGoType(m, hint)
		}
		if _, ok := m["properties"]; ok {
			return g.objectGoType(m, hint)
		}
		return "any", "any"
	}
	switch tt := t.(type) {
	case string:
		if p, k, ok := primitiveGoType(tt); ok {
			switch k {
			case "map":
				return "map[string]any", "map"
			case "slice":
				items, _ := m["items"]
				elem, _ := g.goTypeOf(items, hint+"Item")
				if elem == "any" {
					return "[]any", "slice"
				}
				return "[]" + elem, "slice"
			default:
				return p, k
			}
		}
		// "type" naming another schema type.
		return g.refGoType(tt)
	case []any:
		return g.unionGoType(tt, hint)
	}
	// {}", treat "type" being a nested schema object (unusual) generically.
	if tm, ok := asMap(t); ok {
		_ = tm
		return "any", "any"
	}
	return "any", "any"
}

// unionGoType resolves ["null", T] to *T and other unions to any.
func (g *generator) unionGoType(variants []any, hint string) (string, string) {
	var nonNull []any
	for _, v := range variants {
		if s, ok := v.(string); ok && strings.ToLower(s) == "null" {
			continue
		}
		if m, ok := asMap(v); ok {
			if ts, ok := strVal(m, "$ref"); ok && ts == "" {
				continue
			}
			if ts, ok := m["type"]; ok {
				if s, ok := ts.(string); ok && strings.ToLower(s) == "null" {
					continue
				}
			}
		}
		nonNull = append(nonNull, v)
	}
	if len(nonNull) == 1 {
		inner, kind := g.goTypeOf(nonNull[0], hint)
		if strings.HasPrefix(inner, "*") || inner == "any" {
			return inner, kind
		}
		return "*" + inner, "ptr"
	}
	return "any", "any"
}

// objectGoType handles object schemas; declares a named struct when hint != "".
func (g *generator) objectGoType(m map[string]any, hint string) (string, string) {
	if hint == "" {
		return "map[string]any", "map"
	}
	g.defineStruct(hint, m)
	return hint, "struct"
}

func (g *generator) kindOf(name string) string {
	if k, ok := g.knownKinds[name]; ok {
		return k
	}
	return "any"
}
