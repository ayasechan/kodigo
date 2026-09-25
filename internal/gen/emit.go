package gen

import (
	"fmt"
	"sort"
	"strings"
)

// ---------- definition emitters ----------

func commentFor(desc string, extra ...string) string {
	var lines []string
	if desc != "" {
		for _, l := range strings.Split(desc, "\n") {
			lines = append(lines, "// "+strings.TrimSpace(l))
		}
	}
	for _, e := range extra {
		if e != "" {
			lines = append(lines, "// "+e)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func (g *generator) defineStub(ref string) string {
	name := typeName(ref)
	g.knownKinds[name] = "string"
	return fmt.Sprintf("// %s is a fallback for the unresolved $ref %q.\ntype %s string\n", name, ref, name)
}

// defineTopLevel defines one named type from types.json and returns its source.
func (g *generator) defineTopLevel(schemaName string, node any) string {
	goName := typeName(schemaName)
	if g.defined[goName] {
		return ""
	}
	g.defined[goName] = true
	m, ok := asMap(node)
	if !ok {
		// Should not happen; fall back to any.
		g.knownKinds[goName] = "any"
		return fmt.Sprintf("type %s any\n", goName)
	}
	desc, _ := m["description"].(string)

	// $ref alias: type X = Ref (e.g. param-only aliases).
	if ref, ok := strVal(m, "$ref"); ok {
		target, _ := g.refGoType(ref)
		g.knownKinds[goName] = g.kindOf(target)
		return fmt.Sprintf("%stype %s = %s\n", commentFor(desc, "Alias for "+ref+"."), goName, target)
	}

	// extends without explicit type.
	if _, hasType := m["type"]; !hasType {
		if ext, ok := strVal(m, "extends"); ok {
			if _, hasProps := m["properties"]; hasProps {
				return g.emitStruct(goName, m, desc)
			}
			if items, ok := m["items"]; ok {
				elem, _ := g.goTypeOf(items, goName+"Item")
				g.knownKinds[goName] = "slice"
				return fmt.Sprintf("%stype %s []%s\n", commentFor(desc, "Extends "+ext+"."), goName, elem)
			}
			base, _ := g.refGoType(ext)
			g.knownKinds[goName] = g.kindOf(base)
			return fmt.Sprintf("%stype %s = %s\n", commentFor(desc, "Extends "+ext+"."), goName, base)
		}
		if _, ok := m["properties"]; ok {
			return g.emitStruct(goName, m, desc)
		}
		g.knownKinds[goName] = "any"
		return fmt.Sprintf("%stype %s any\n", commentFor(desc), goName)
	}

	t := m["type"]
	switch tt := t.(type) {
	case string:
		if _, k, ok := primitiveGoType(tt); ok {
			switch k {
			case "string":
				if enum, ok := m["enum"].([]any); ok {
					return g.emitStringEnum(goName, enum, desc)
				}
				g.knownKinds[goName] = "string"
				def := defaultComment(m)
				return fmt.Sprintf("%stype %s string\n", commentFor(desc, def), goName)
			case "map":
				return g.emitStruct(goName, m, desc)
			case "slice":
				items, _ := m["items"]
				elem, _ := g.goTypeOf(items, goName+"Item")
				g.knownKinds[goName] = "slice"
				return fmt.Sprintf("%stype %s []%s\n", commentFor(desc), goName, elem)
			default:
				g.knownKinds[goName] = k
				def := defaultComment(m)
				return fmt.Sprintf("%stype %s %s\n", commentFor(desc, def), goName, mapKindGo(k))
			}
		}
		// "type" is another type's name.
		target, _ := g.refGoType(tt)
		g.knownKinds[goName] = g.kindOf(target)
		return fmt.Sprintf("%stype %s = %s\n", commentFor(desc), goName, target)
	case []any:
		// Union named type.
		nonNull := nonNullVariants(tt)
		if len(nonNull) == 1 {
			// Single non-null variant.
			if vm, ok := asMap(nonNull[0]); ok {
				if _, hasProps := vm["properties"]; hasProps {
					return g.emitStruct(goName, vm, desc)
				}
				if t, _ := vm["type"]; t == "array" {
					elem, _ := g.namedValueType(vm["items"], goName+"Item")
					g.knownKinds[goName] = "slice"
					return fmt.Sprintf("%stype %s []%s\n", commentFor(desc), goName, elem)
				}
			}
			inner, kind := g.goTypeOf(nonNull[0], "")
			switch kind {
			case "struct", "slice", "map":
				// goTypeOf already declared goName via hint.
				return ""
			default:
				g.knownKinds[goName] = kind
				return fmt.Sprintf("%stype %s %s\n", commentFor(desc, "Nullable "+inner+"."), goName, inner)
			}
		}
		g.knownKinds[goName] = "any"
		return fmt.Sprintf("%s// %s is a union; decoded as any. See schema key %q in types.json.\ntype %s any\n",
			commentFor(desc), goName, schemaName, goName)
	}
	g.knownKinds[goName] = "any"
	return fmt.Sprintf("%stype %s any\n", commentFor(desc), goName)
}

func mapKindGo(k string) string {
	switch k {
	case "int":
		return "int"
	case "float":
		return "float64"
	case "bool":
		return "bool"
	}
	return "any"
}

func defaultComment(m map[string]any) string {
	if d, ok := m["default"]; ok {
		return fmt.Sprintf("Default: %s.", fmtDefault(d))
	}
	return ""
}

// fmtDefault renders a schema default value for doc comments.
func fmtDefault(v any) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprintf("%v", v)
}

func nonNullVariants(variants []any) []any {
	var out []any
	for _, v := range variants {
		if s, ok := v.(string); ok && strings.ToLower(s) == "null" {
			continue
		}
		if m, ok := asMap(v); ok {
			if ts, ok := m["type"]; ok {
				if s, ok := ts.(string); ok && strings.ToLower(s) == "null" {
					continue
				}
			}
		}
		out = append(out, v)
	}
	return out
}

func (g *generator) emitStringEnum(goName string, enum []any, desc string) string {
	g.knownKinds[goName] = "string"
	var b strings.Builder
	b.WriteString(commentFor(desc))
	fmt.Fprintf(&b, "type %s string\n\n", goName)
	b.WriteString("const (\n")
	for _, e := range enum {
		s, ok := e.(string)
		if !ok {
			continue
		}
		c := g.uniqueConst(enumConstName(goName, s))
		fmt.Fprintf(&b, "\t%s %s = %q\n", c, goName, s)
	}
	b.WriteString(")\n")
	return b.String()
}

// emitStruct emits "type Name struct {...}" for an object schema node.
func (g *generator) emitStruct(goName string, m map[string]any, desc string) string {
	g.knownKinds[goName] = "struct"
	var b strings.Builder
	b.WriteString(commentFor(desc))
	fmt.Fprintf(&b, "type %s struct {\n", goName)
	b.WriteString(g.structBody(goName, m))
	b.WriteString("}\n")
	return b.String()
}

// defineStruct declares a named struct immediately (used for inline objects).
func (g *generator) defineStruct(goName string, m map[string]any) {
	if g.defined[goName] {
		return
	}
	g.defined[goName] = true
	desc, _ := m["description"].(string)
	var b strings.Builder
	b.WriteString(commentFor(desc))
	fmt.Fprintf(&b, "type %s struct {\n", goName)
	b.WriteString(g.structBody(goName, m))
	b.WriteString("}\n")
	g.knownKinds[goName] = "struct"
	g.extraDefs = append(g.extraDefs, b.String())
}

// structBody renders struct fields for an object schema's properties.
func (g *generator) structBody(parent string, m map[string]any) string {
	var b strings.Builder
	if ext, ok := strVal(m, "extends"); ok {
		base, _ := g.refGoType(ext)
		fmt.Fprintf(&b, "\t%s // embedded base type %s\n", base, ext)
	}
	props, _ := m["properties"].(map[string]any)
	names := make([]string, 0, len(props))
	for n := range props {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, jsonName := range names {
		b.WriteString(g.structField(parent, jsonName, props[jsonName]))
	}
	return b.String()
}

// structField renders one struct field, defining nested named types as needed.
func (g *generator) structField(parent, jsonName string, schema any) string {
	goF := fieldName(jsonName)
	m, _ := asMap(schema)
	required := m != nil && boolVal(m, "required")
	hint := parent + goF
	typ, kind := g.namedValueType(schema, hint)
	tag := jsonName
	if !required {
		tag += ",omitempty"
		if kind == "string" || kind == "int" || kind == "float" || kind == "bool" || kind == "struct" {
			if !strings.HasPrefix(typ, "*") && !strings.HasPrefix(typ, "[]") && !strings.HasPrefix(typ, "map[") && typ != "any" {
				typ = "*" + typ
			}
		}
	}
	var doc string
	if m != nil {
		desc, _ := m["description"].(string)
		var extras []string
		if d, ok := m["default"]; ok && !required {
			extras = append(extras, fmt.Sprintf("Default: %s.", fmtDefault(d)))
		}
		if e, ok := m["enum"].([]any); ok && kind != "string" {
			extras = append(extras, "Enum: "+enumList(e)+".")
		}
		doc = commentFor(desc, extras...)
	}
	if doc != "" {
		lines := strings.Split(strings.TrimSuffix(doc, "\n"), "\n")
		var out strings.Builder
		for _, l := range lines {
			out.WriteString("\t" + l + "\n")
		}
		doc = out.String()
	}
	return fmt.Sprintf("%s\t%s %s `json:\"%s\"`\n", doc, goF, typ, tag)
}

// namedValueType is goTypeOf but ensures anonymous objects/arrays-of-objects
// get stable named declarations instead of inline structs.
func (g *generator) namedValueType(schema any, hint string) (string, string) {
	if ref, ok := refName(schema); ok {
		return g.refGoType(ref)
	}
	m, ok := asMap(schema)
	if !ok {
		return g.goTypeOf(schema, hint)
	}
	t, hasType := m["type"]
	if s, ok := t.(string); ok && hasType {
		if _, k, ok := primitiveGoType(s); ok {
			switch k {
			case "map":
				// Anonymous object: declare named struct if it has properties.
				if _, hasProps := m["properties"]; hasProps {
					if !g.defined[hint] {
						g.defineStruct(hint, m)
					}
					return hint, "struct"
				}
				return "map[string]any", "map"
			case "slice":
				items, _ := m["items"]
				if im, ok := asMap(items); ok {
					if _, isRef := strVal(im, "$ref"); !isRef {
						if _, hasProps := im["properties"]; hasProps {
							itemName := hint + "Item"
							if !g.defined[itemName] {
								g.defineStruct(itemName, im)
							}
							return "[]" + itemName, "slice"
						}
					}
				}
				return g.goTypeOf(schema, "")
			}
		}
	}
	return g.goTypeOf(schema, hint)
}

func enumList(e []any) string {
	var parts []string
	for _, v := range e {
		parts = append(parts, fmt.Sprintf("%v", v))
	}
	return strings.Join(parts, ", ")
}
