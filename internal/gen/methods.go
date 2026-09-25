package gen

import (
	"fmt"
	"path/filepath"
	"strings"
)

func methodGoName(full string) string {
	parts := strings.SplitN(full, ".", 2)
	if len(parts) == 2 {
		return exportName(parts[0]) + exportName(parts[1])
	}
	return exportName(full)
}

func (g *generator) genMethods() (string, error) {
	var b strings.Builder
	b.WriteString(g.fileHeader(fmt.Sprintf("// Schema version %s.\n\n", g.version), filepath.Join(g.schemaDir, "methods.json")))
	b.WriteString("import \"context\"\n\n")
	b.WriteString("// JSON-RPC method names.\nconst (\n")
	for _, full := range g.methodKeys {
		fmt.Fprintf(&b, "\tMethod%s = %q\n", methodGoName(full), full)
	}
	b.WriteString(")\n\n")

	for _, full := range g.methodKeys {
		node := decodeNode(g.methodRaw[full])
		m, ok := asMap(node)
		if !ok {
			continue
		}
		goName := methodGoName(full)
		desc, _ := m["description"].(string)

		// Params.
		var paramNodes []any
		if p, ok := m["params"].([]any); ok {
			paramNodes = p
		}
		paramsType := ""
		if len(paramNodes) > 0 {
			paramsType = goName + "Params"
			var pb strings.Builder
			fmt.Fprintf(&pb, "%stype %s struct {\n", commentFor("Params for "+full+"."), paramsType)
			g.knownKinds[paramsType] = "struct"
			g.defined[paramsType] = true
			for _, p := range paramNodes {
				pm, ok := asMap(p)
				if !ok {
					continue
				}
				pname, _ := strVal(pm, "name")
				if pname == "" {
					continue
				}
				// The field schema is the param node minus "name".
				fs := map[string]any{}
				for k, v := range pm {
					if k != "name" {
						fs[k] = v
					}
				}
				pb.WriteString(g.structField(paramsType, pname, fs))
			}
			pb.WriteString("}\n\n")
			g.extraDefs = append(g.extraDefs, pb.String())
		}

		// Result.
		var resultType string
		if r, ok := m["returns"]; ok && r != nil {
			if s, ok := r.(string); ok {
				if p, k, ok := primitiveGoType(s); ok && k != "map" && k != "slice" {
					resultType = p
				} else if k == "map" || k == "slice" {
					resultType = p
				} else {
					t, _ := g.refGoType(s)
					resultType = t
				}
			} else if rm, ok := asMap(r); ok {
				if ref, ok := strVal(rm, "$ref"); ok {
					t, _ := g.refGoType(ref)
					resultType = t
				} else {
					resultType = goName + "Result"
					if !g.defined[resultType] {
						g.defined[resultType] = true
						def := g.defineResultType(resultType, rm, full)
						g.extraDefs = append(g.extraDefs, def)
					}
				}
			}
		}

		b.WriteString(g.extraDefsFlush())
		b.WriteString(commentFor(desc, full+"."))
		if paramsType == "" && resultType == "" {
			fmt.Fprintf(&b, "func (c *Client) %s(ctx context.Context) error {\n\treturn c.Call(ctx, Method%s, nil, nil)\n}\n\n", goName, goName)
		} else if paramsType == "" {
			fmt.Fprintf(&b, "func (c *Client) %s(ctx context.Context) (%s, error) {\n\tvar result %s\n\terr := c.Call(ctx, Method%s, nil, &result)\n\treturn result, err\n}\n\n", goName, resultType, resultType, goName)
		} else if resultType == "" {
			fmt.Fprintf(&b, "func (c *Client) %s(ctx context.Context, params %s) error {\n\treturn c.Call(ctx, Method%s, params, nil)\n}\n\n", goName, paramsType, goName)
		} else {
			fmt.Fprintf(&b, "func (c *Client) %s(ctx context.Context, params %s) (%s, error) {\n\tvar result %s\n\terr := c.Call(ctx, Method%s, params, &result)\n\treturn result, err\n}\n\n", goName, paramsType, resultType, resultType, goName)
		}
	}
	b.WriteString(g.extraDefsFlush())
	return b.String(), nil
}

// extraDefsFlush returns pending definitions and clears the queue.
func (g *generator) extraDefsFlush() string {
	if len(g.extraDefs) == 0 {
		return ""
	}
	s := strings.Join(g.extraDefs, "\n") + "\n"
	g.extraDefs = nil
	return s
}

// defineResultType declares a named result type for a method returns schema.
func (g *generator) defineResultType(name string, m map[string]any, full string) string {
	t, _ := m["type"]
	if s, ok := t.(string); ok {
		if _, k, ok := primitiveGoType(s); ok {
			switch k {
			case "map":
				return g.emitStruct(name, m, "Result for "+full+".")
			case "slice":
				items, _ := m["items"]
				elem, _ := g.namedValueType(items, name+"Item")
				g.knownKinds[name] = "slice"
				return fmt.Sprintf("// Result for %s.\ntype %s []%s\n", full, name, elem)
			case "string":
				if enum, ok := m["enum"].([]any); ok {
					desc := "Result for " + full + "."
					return g.emitStringEnum(name, enum, desc)
				}
				g.knownKinds[name] = "string"
				return fmt.Sprintf("// Result for %s.\ntype %s string\n", full, name)
			default:
				g.knownKinds[name] = k
				return fmt.Sprintf("// Result for %s.\ntype %s %s\n", full, name, mapKindGo(k))
			}
		}
		target, _ := g.refGoType(s)
		g.knownKinds[name] = g.kindOf(target)
		return fmt.Sprintf("// Result for %s.\ntype %s = %s\n", full, name, target)
	}
	if arr, ok := t.([]any); ok {
		nonNull := nonNullVariants(arr)
		if len(nonNull) == 1 {
			inner, kind := g.namedValueType(nonNull[0], name)
			if kind == "struct" || kind == "slice" {
				return "" // already declared under name
			}
			g.knownKinds[name] = kind
			_ = inner
			// Union of one: alias the single variant.
			inner2, _ := g.goTypeOf(nonNull[0], "")
			return fmt.Sprintf("// Result for %s.\ntype %s = %s\n", full, name, inner2)
		}
		g.knownKinds[name] = "any"
		return fmt.Sprintf("// Result for %s (union, decoded as any).\ntype %s any\n", full, name)
	}
	// No explicit type but properties/extends present.
	if _, ok := m["properties"]; ok {
		return g.emitStruct(name, m, "Result for "+full+".")
	}
	if ext, ok := strVal(m, "extends"); ok {
		base, _ := g.refGoType(ext)
		g.knownKinds[name] = g.kindOf(base)
		return fmt.Sprintf("// Result for %s (extends %s).\ntype %s = %s\n", full, ext, name, base)
	}
	g.knownKinds[name] = "any"
	return fmt.Sprintf("// Result for %s.\ntype %s any\n", full, name)
}
