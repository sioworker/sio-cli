package sio

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed lang/*.json
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

func merge(b []byte, err error) {
	if err == nil {
		json.Unmarshal(b, &L)
	}
}

func LoadLang(code string) { // en -> built-in code -> ~/.config/sio/lang/code.json
	merge(langFS.ReadFile("lang/en.json"))
	merge(langFS.ReadFile("lang/" + code + ".json"))
	merge(os.ReadFile(filepath.Join(filepath.Dir(CfgPath()), "lang", code+".json")))
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
