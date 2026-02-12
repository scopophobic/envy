package dotenv

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Serialize returns .env file content for the given key/values.
func Serialize(values map[string]string) []byte {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b bytes.Buffer
	for _, k := range keys {
		v := values[k]
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(escapeValue(v))
		b.WriteString("\n")
	}
	return b.Bytes()
}

func escapeValue(v string) string {
	// If value contains spaces/newlines/# or quotes, wrap in double quotes and escape.
	needsQuotes := strings.ContainsAny(v, " \t\r\n#\"")
	if !needsQuotes {
		return v
	}
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	v = strings.ReplaceAll(v, "\n", `\n`)
	return `"` + v + `"`
}

// EnsureGitignoreHasDotenv adds ".env" line to .gitignore in the given dir.
func EnsureGitignoreHasDotenv(dir string) error {
	p := filepath.Join(dir, ".gitignore")
	line := ".env"

	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(p, []byte(line+"\n"), 0o644)
		}
		return err
	}

	content := string(b)
	for _, l := range strings.Split(content, "\n") {
		if strings.TrimSpace(l) == line {
			return nil
		}
	}

	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += line + "\n"
	return os.WriteFile(p, []byte(content), 0o644)
}

// WriteEnvFile writes `.env` in dir, overwriting any existing file.
func WriteEnvFile(dir string, values map[string]string) (string, error) {
	p := filepath.Join(dir, ".env")
	if err := os.WriteFile(p, Serialize(values), 0o600); err != nil {
		return "", err
	}
	return p, nil
}

// LoadEnvFile loads key/value pairs from a .env file (minimal parser).
func LoadEnvFile(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	out := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" {
			continue
		}
		out[k] = unescapeValue(v)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("failed reading .env: %w", err)
	}
	return out, nil
}

func unescapeValue(v string) string {
	if len(v) >= 2 && strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`) {
		v = strings.TrimSuffix(strings.TrimPrefix(v, `"`), `"`)
		v = strings.ReplaceAll(v, `\n`, "\n")
		v = strings.ReplaceAll(v, `\"`, `"`)
		v = strings.ReplaceAll(v, `\\`, `\`)
		return v
	}
	return v
}

