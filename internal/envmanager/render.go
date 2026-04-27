package envmanager

import (
	"fmt"
	"sort"
	"strings"
)

func RenderDotenv(values map[string]string) (string, error) {
	keys := sortedKeys(values)

	var builder strings.Builder
	for _, key := range keys {
		if err := ValidateKey(key); err != nil {
			return "", err
		}
		builder.WriteString(fmt.Sprintf("%s=%s\n", key, quoteDotenv(values[key])))
	}

	return builder.String(), nil
}

func RenderExports(values map[string]string) (string, error) {
	keys := sortedKeys(values)

	var builder strings.Builder
	for _, key := range keys {
		if err := ValidateKey(key); err != nil {
			return "", err
		}
		builder.WriteString(fmt.Sprintf("export %s=%s\n", key, quoteShell(values[key])))
	}

	return builder.String(), nil
}

func ParseDotenv(data []byte) (map[string]string, error) {
	values := map[string]string{}
	lines := strings.Split(string(data), "\n")

	for lineNumber := 0; lineNumber < len(lines); lineNumber++ {
		rawLine := lines[lineNumber]
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf(".env line %d must look like KEY=value", lineNumber+1)
		}

		key = strings.TrimSpace(key)
		if err := ValidateKey(key); err != nil {
			return nil, fmt.Errorf(".env line %d: %w", lineNumber+1, err)
		}

		valueText := strings.TrimSpace(rawValue)
		if startsQuotedValue(valueText) && !hasClosingQuote(valueText, valueText[0]) {
			startLine := lineNumber + 1
			for lineNumber+1 < len(lines) {
				lineNumber++
				valueText += "\n" + lines[lineNumber]
				if hasClosingQuote(valueText, valueText[0]) {
					break
				}
			}
			if !hasClosingQuote(valueText, valueText[0]) {
				return nil, fmt.Errorf(".env line %d: unterminated quoted value", startLine)
			}
		}

		value, err := parseDotenvValue(valueText)
		if err != nil {
			return nil, fmt.Errorf(".env line %d: %w", lineNumber+1, err)
		}
		values[key] = value
	}

	return values, nil
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func quoteDotenv(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\':
			builder.WriteString(`\\`)
		case '"':
			builder.WriteString(`\"`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			builder.WriteRune(r)
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

func quoteShell(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func startsQuotedValue(raw string) bool {
	return strings.HasPrefix(raw, `"`) || strings.HasPrefix(raw, `'`)
}

func hasClosingQuote(raw string, quote byte) bool {
	if len(raw) < 2 || raw[0] != quote {
		return false
	}

	escaped := false
	for index := 1; index < len(raw); index++ {
		if quote == '"' && escaped {
			escaped = false
			continue
		}
		if quote == '"' && raw[index] == '\\' {
			escaped = true
			continue
		}
		if raw[index] == quote {
			return true
		}
	}
	return false
}

func parseDotenvValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	if strings.HasPrefix(raw, `"`) {
		end := closingQuoteIndex(raw, '"')
		if end < 0 {
			return "", fmt.Errorf("unterminated double-quoted value")
		}
		return unescapeDoubleQuotedDotenv(raw[1:end])
	}

	if strings.HasPrefix(raw, `'`) {
		end := closingQuoteIndex(raw, '\'')
		if end < 0 {
			return "", fmt.Errorf("unterminated single-quoted value")
		}
		return raw[1:end], nil
	}

	if index := strings.Index(raw, " #"); index >= 0 {
		raw = raw[:index]
	}
	return strings.TrimSpace(raw), nil
}

func closingQuoteIndex(raw string, quote byte) int {
	escaped := false
	for index := 1; index < len(raw); index++ {
		if quote == '"' && escaped {
			escaped = false
			continue
		}
		if quote == '"' && raw[index] == '\\' {
			escaped = true
			continue
		}
		if raw[index] == quote {
			return index
		}
	}
	return -1
}

func unescapeDoubleQuotedDotenv(raw string) (string, error) {
	var builder strings.Builder
	for index := 0; index < len(raw); index++ {
		if raw[index] != '\\' {
			builder.WriteByte(raw[index])
			continue
		}

		index++
		if index >= len(raw) {
			return "", fmt.Errorf("unfinished escape sequence")
		}

		switch raw[index] {
		case 'n':
			builder.WriteByte('\n')
		case 'r':
			builder.WriteByte('\r')
		case 't':
			builder.WriteByte('\t')
		case '\\':
			builder.WriteByte('\\')
		case '"':
			builder.WriteByte('"')
		default:
			builder.WriteByte(raw[index])
		}
	}
	return builder.String(), nil
}
