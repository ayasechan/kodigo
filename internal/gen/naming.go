package gen

import (
	"fmt"
	"strings"
)

// ---------- naming ----------

var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// exportName converts identifiers like "Player.Repeat", "playerid",
// "showtitle" into exported Go names like "PlayerRepeat", "PlayerID".
func exportName(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || r == ' ' || r == '/' || r == ':'
	})
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	name := b.String()
	// Common initialism fixes (interior occurrences).
	fixes := []struct{ old, new string }{
		{"Url", "URL"}, {"Uri", "URI"},
		{"Tv", "TV"}, {"Pvr", "PVR"}, {"Gui", "GUI"}, {"Osd", "OSD"},
		{"Jsonrpc", "JSONRPC"}, {"Mpaa", "MPAA"}, {"Imdbnumber", "IMDbNumber"},
		{"Dvd", "DVD"}, {"Cd", "CD"}, {"Hd", "HD"}, {"Usp", "USP"},
	}
	for _, f := range fixes {
		name = strings.ReplaceAll(name, f.old, f.new)
	}
	// Trailing-id/url suffixes: songid -> SongID, channeluid -> ChannelUID.
	// All such names in the Kodi schema are genuine identifiers/URLs.
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "uid"):
		name = name[:len(name)-3] + "UID"
	case strings.HasSuffix(lower, "ids"):
		name = name[:len(name)-3] + "IDs"
	case strings.HasSuffix(lower, "id"):
		name = name[:len(name)-2] + "ID"
	case strings.HasSuffix(lower, "url"):
		name = name[:len(name)-3] + "URL"
	case strings.HasSuffix(lower, "uri"):
		name = name[:len(name)-3] + "URI"
	}
	if name == "" {
		name = "X"
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "N" + name
	}
	// Exported names start with an uppercase letter and can never collide
	// with a keyword; only guard exact lowercase matches (defensive).
	if goKeywords[name] {
		name += "_"
	}
	return name
}

// typeName converts a schema type name ("Player.Repeat") to a Go type name.
func typeName(s string) string { return exportName(s) }

// fieldName converts a JSON property name to a Go struct field name.
func fieldName(s string) string {
	if s == "id" {
		return "ID"
	}
	return exportName(s)
}

// enumConstName builds a const name for an enum value.
func enumConstName(typeGo, value string) string {
	v := exportName(strings.ToLower(value))
	if v == "" {
		v = "Empty"
	}
	return typeGo + v
}

func (g *generator) uniqueConst(base string) string {
	name := base
	for i := 2; g.usedConsts[name]; i++ {
		name = fmt.Sprintf("%s%d", base, i)
	}
	g.usedConsts[name] = true
	return name
}
