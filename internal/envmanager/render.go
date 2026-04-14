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
