package gen

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func (g *generator) genNotifications() (string, error) {
	var b strings.Builder
	b.WriteString(g.fileHeader(fmt.Sprintf("// Schema version %s.\n\n", g.version), filepath.Join(g.schemaDir, "notifications.json")))
	b.WriteString("// Notification names.\nconst (\n")
	for _, full := range g.notifKeys {
		fmt.Fprintf(&b, "\tNotification%s = %q\n", methodGoName(full), full)
	}
	b.WriteString(")\n\n")

	for _, full := range g.notifKeys {
		node := decodeNode(g.notifRaw[full])
		m, ok := asMap(node)
		if !ok {
			continue
		}
		goName := methodGoName(full)
		desc, _ := m["description"].(string)
		b.WriteString(commentFor(desc, full+"."))

		var params []any
		if p, ok := m["params"].([]any); ok {
			params = p
		}
		// Find the "data" param.
		var dataSchema any
		for _, p := range params {
			pm, ok := asMap(p)
			if !ok {
				continue
			}
			if n, _ := strVal(pm, "name"); n == "data" {
				fs := map[string]any{}
				for k, v := range pm {
					if k != "name" {
						fs[k] = v
					}
				}
				dataSchema = fs
			}
		}
		dataType := ""
		if dataSchema != nil {
			if ref, ok := refName(dataSchema); ok {
				target, _ := g.refGoType(ref)
				dataType = goName + "Data"
				if !g.defined[dataType] {
					g.defined[dataType] = true
					g.knownKinds[dataType] = g.kindOf(target)
					g.extraDefs = append(g.extraDefs,
						fmt.Sprintf("// %s carries the data of %s.\ntype %s = %s\n", dataType, full, dataType, target))
				}
			} else if dm, ok := asMap(dataSchema); ok {
				if _, hasProps := dm["properties"]; hasProps {
					if t, _ := dm["type"]; t == nil || t == "object" {
						dataType = goName + "Data"
						if !g.defined[dataType] {
							g.defined[dataType] = true
							g.extraDefs = append(g.extraDefs,
								g.emitStruct(dataType, dm, "Data for "+full+"."))
						}
					} else {
						dataType = goName + "Data"
						if !g.defined[dataType] {
							g.defineGenericAlias(dataType, dm, full)
						}
					}
				} else {
					typ, kind := g.namedValueType(dataSchema, goName+"Data")
					_ = kind
					dataType = typ
				}
			}
		}
		b.WriteString(g.extraDefsFlush())
		if dataType != "" && !strings.HasPrefix(dataType, "*") && !strings.HasPrefix(dataType, "[]") && !strings.HasPrefix(dataType, "map[") && dataType != "any" && dataType != "string" && dataType != "int" && dataType != "bool" && dataType != "float64" {
			fmt.Fprintf(&b, "// %sHandler handles %s notifications.\ntype %sHandler func(sender string, data %s)\n\n", goName, full, goName, dataType)
		} else if dataType != "" {
			fmt.Fprintf(&b, "// %sData carries the data of %s.\n//\n// Schema type: %s.\n\n", goName, full, dataType)
		} else {
			b.WriteString("\n")
		}
	}
	// Stubs for refs first seen via methods/notifications.
	if len(g.missing) > 0 {
		var names []string
		for n := range g.missing {
			names = append(names, n)
		}
		sort.Strings(names)
		b.WriteString("// Unresolved references (not present in schema/types.json).\n")
		for _, n := range names {
			if !g.defined[typeName(n)] {
				g.defined[typeName(n)] = true
				b.WriteString(g.defineStub(n))
				b.WriteString("\n")
			}
		}
	}
	return b.String(), nil
}

func (g *generator) defineGenericAlias(name string, m map[string]any, full string) {
	typ, kind := g.goTypeOf(m, "")
	g.defined[name] = true
	g.knownKinds[name] = kind
	g.extraDefs = append(g.extraDefs,
		fmt.Sprintf("// %s carries the data of %s.\ntype %s = %s\n", name, full, name, typ))
}
