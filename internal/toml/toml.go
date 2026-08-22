package toml

import (
	"fmt"
	"strconv"
	"strings"
)

type Table struct {
	keys   []string
	values map[string]interface{}
}

func NewTable() *Table {
	return &Table{values: make(map[string]interface{})}
}

func (t *Table) Set(key string, value interface{}) {
	if t.values == nil {
		t.values = make(map[string]interface{})
	}
	if _, exists := t.values[key]; !exists {
		t.keys = append(t.keys, key)
	}
	t.values[key] = value
}

func (t *Table) Keys() []string {
	return append([]string(nil), t.keys...)
}

func (t *Table) Get(key string) (interface{}, bool) {
	if t.values == nil {
		return nil, false
	}
	v, ok := t.values[key]
	return v, ok
}

func (t *Table) GetString(key, def string) string {
	if v, ok := t.Get(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

func (t *Table) GetBool(key string, def bool) bool {
	if v, ok := t.Get(key); ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func (t *Table) GetTable(key string) (*Table, bool) {
	if v, ok := t.Get(key); ok {
		if sub, ok := v.(*Table); ok {
			return sub, true
		}
	}
	return nil, false
}

func (t *Table) GetStringSlice(key string) []string {
	if v, ok := t.Get(key); ok {
		if arr, ok := v.([]string); ok {
			return arr
		}
	}
	return nil
}

func (t *Table) SubTable(key string) *Table {
	if sub, ok := t.GetTable(key); ok {
		return sub
	}
	sub := NewTable()
	t.Set(key, sub)
	return sub
}

type Document struct {
	Root *Table
}

func NewDocument() *Document {
	return &Document{Root: NewTable()}
}

func Parse(src string) (*Document, error) {
	doc := NewDocument()
	cur := doc.Root

	lines := strings.Split(src, "\n")
	for lineNo, raw := range lines {
		line := stripComment(raw)
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") {
			header, err := parseHeader(line)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
			}
			cur = doc.Root
			for _, part := range header {
				cur = cur.SubTable(part)
			}
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected key = value, got %q", lineNo+1, line)
		}
		key = strings.TrimSpace(key)
		key = unquoteKey(key)
		val = strings.TrimSpace(val)

		parsed, err := parseValue(val)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
		}
		cur.Set(key, parsed)
	}

	return doc, nil
}

func unquoteKey(k string) string {
	if len(k) >= 2 && (k[0] == '"' || k[0] == '\'') && k[len(k)-1] == k[0] {
		return k[1 : len(k)-1]
	}
	return k
}

func stripComment(line string) string {
	inString := false
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		if inString {
			if c == quote && (i == 0 || line[i-1] != '\\') {
				inString = false
			}
			continue
		}
		if c == '"' || c == '\'' {
			inString = true
			quote = c
			continue
		}
		if c == '#' {
			return line[:i]
		}
	}
	return line
}

func parseHeader(line string) ([]string, error) {
	if !strings.HasSuffix(line, "]") {
		return nil, fmt.Errorf("malformed table header %q", line)
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return nil, fmt.Errorf("empty table header")
	}
	parts := strings.Split(inner, ".")
	for i, p := range parts {
		parts[i] = unquoteKey(strings.TrimSpace(p))
	}
	return parts, nil
}

func parseValue(val string) (interface{}, error) {
	val = strings.TrimSpace(val)
	if val == "" {
		return "", fmt.Errorf("empty value")
	}

	switch val {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	if strings.HasPrefix(val, "[") {
		return parseArray(val)
	}

	if strings.HasPrefix(val, "\"") || strings.HasPrefix(val, "'") {
		return parseString(val)
	}

	if i, err := strconv.ParseInt(val, 10, 64); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return f, nil
	}

	return val, nil
}

func parseString(val string) (string, error) {
	if len(val) < 2 {
		return "", fmt.Errorf("malformed string %q", val)
	}
	quote := val[0]
	if val[len(val)-1] != quote {
		return "", fmt.Errorf("unterminated string %q", val)
	}
	inner := val[1 : len(val)-1]
	if quote == '\'' {
		return inner, nil
	}
	var sb strings.Builder
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c == '\\' && i+1 < len(inner) {
			next := inner[i+1]
			switch next {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			default:
				sb.WriteByte(next)
			}
			i++
			continue
		}
		sb.WriteByte(c)
	}
	return sb.String(), nil
}

func parseArray(val string) ([]string, error) {
	if !strings.HasSuffix(val, "]") {
		return nil, fmt.Errorf("malformed array %q", val)
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(val, "["), "]")
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return []string{}, nil
	}
	var result []string
	for _, part := range splitTopLevel(inner, ',') {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		s, err := parseValueAsString(part)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func parseValueAsString(part string) (string, error) {
	if strings.HasPrefix(part, "\"") || strings.HasPrefix(part, "'") {
		return parseString(part)
	}
	return part, nil
}

func splitTopLevel(s string, sep rune) []string {
	var parts []string
	var cur strings.Builder
	inString := false
	var quote rune
	for _, c := range s {
		if inString {
			cur.WriteRune(c)
			if c == quote {
				inString = false
			}
			continue
		}
		if c == '"' || c == '\'' {
			inString = true
			quote = c
			cur.WriteRune(c)
			continue
		}
		if c == sep {
			parts = append(parts, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteRune(c)
	}
	parts = append(parts, cur.String())
	return parts
}

func Write(doc *Document) string {
	var sb strings.Builder
	writeTable(&sb, doc.Root, nil)
	return sb.String()
}

func writeTable(sb *strings.Builder, t *Table, path []string) {
	var scalarKeys []string
	var tableKeys []string
	for _, k := range t.keys {
		v := t.values[k]
		if _, isTable := v.(*Table); isTable {
			tableKeys = append(tableKeys, k)
		} else {
			scalarKeys = append(scalarKeys, k)
		}
	}

	if len(path) > 0 && (len(scalarKeys) > 0 || len(tableKeys) == 0) {
		fmt.Fprintf(sb, "[%s]\n", strings.Join(path, "."))
	}

	for _, k := range scalarKeys {
		fmt.Fprintf(sb, "%s = %s\n", quoteKeyIfNeeded(k), formatValue(t.values[k]))
	}
	if len(scalarKeys) > 0 && len(tableKeys) > 0 {
		sb.WriteString("\n")
	}

	for i, k := range tableKeys {
		sub := t.values[k].(*Table)
		writeTable(sb, sub, append(append([]string(nil), path...), k))
		if i < len(tableKeys)-1 {
			sb.WriteString("\n")
		}
	}
}

func quoteKeyIfNeeded(k string) string {
	for _, c := range k {
		if !(c == '_' || c == '-' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return strconv.Quote(k)
		}
	}
	return k
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return strconv.Quote(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'g', -1, 64)
	case []string:
		quoted := make([]string, len(val))
		for i, s := range val {
			quoted[i] = strconv.Quote(s)
		}
		return "[" + strings.Join(quoted, ", ") + "]"
	default:
		return strconv.Quote(fmt.Sprintf("%v", val))
	}
}
