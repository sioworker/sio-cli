package sio

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed lang/*.jsonc
var langFS embed.FS

var L = map[string]string{}

func LangCode() string {
	for _, k := range []string{"SIO_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" {
			v, _, _ = strings.Cut(v, ".")
			v, _, _ = strings.Cut(v, "_")
			if v == "C" || v == "POSIX" {
				return "en"
			}
			return strings.ToLower(v)
		}
	}
	return "en"
}

func jsonc(b []byte) []byte { // drop // and /* */ comments, then trailing commas
	o, i, n, str := []byte{}, 0, len(b), false
	for i < n {
		c := b[i]
		if str {
			if c == '\\' && i+1 < n {
				o = append(o, c)
				i++
				c = b[i]
			} else if c == '"' {
				str = false
			}
		} else if c == '"' {
			str = true
		} else if c == '/' && i+1 < n && b[i+1] == '/' {
			for i < n && b[i] != '\n' {
				i++
			}
			continue
		} else if c == '/' && i+1 < n && b[i+1] == '*' {
			i += 2
			for i+1 < n && !(b[i] == '*' && b[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}
		o = append(o, c)
		i++
	}
	b, o, i, n, str = o, []byte{}, 0, len(o), false
	for i < n {
		c := b[i]
		if str {
			if c == '\\' && i+1 < n {
				o = append(o, c)
				i++
				c = b[i]
			} else if c == '"' {
				str = false
			}
		} else if c == '"' {
			str = true
		} else if c == ',' {
			j := i + 1
			for j < n && (b[j] == ' ' || b[j] == '\t' || b[j] == '\r' || b[j] == '\n') {
				j++
			}
			if j < n && (b[j] == '}' || b[j] == ']') {
				i++
				continue
			}
		}
		o = append(o, c)
		i++
	}
	return o
}

func merge(b []byte, err error) {
	if err == nil {
		json.Unmarshal(jsonc(b), &L)
	}
}

func LoadLang(code string) { // en -> built-in code -> ~/.config/sio/lang/code.jsonc
	merge(langFS.ReadFile("lang/en.jsonc"))
	merge(langFS.ReadFile("lang/" + code + ".jsonc"))
	merge(os.ReadFile(filepath.Join(filepath.Dir(CfgPath()), "lang", code+".jsonc")))
}

func T(k string, a ...any) string {
	s, ok := L[k]
	if !ok {
		s = k
	}
	if len(a) == 0 {
		return s
	}
	return fmt.Sprintf(s, a...)
}
