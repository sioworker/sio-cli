package sio

import (
	"encoding/json"
	"strings"
	"testing"
)

func verbs(s string) int { return strings.Count(s, "%") - 2*strings.Count(s, "%%") }

func TestJsonc(t *testing.T) {
	in := `// top
{
	"a": "x // not a comment", /* block */
	"b": "q\"/*", // tricky
	"c": [1, 2,],
}`
	m := map[string]any{}
	if err := json.Unmarshal(jsonc([]byte(in)), &m); err != nil {
		t.Fatal(err, string(jsonc([]byte(in))))
	}
	if m["a"] != "x // not a comment" || m["b"] != `q"/*` || len(m["c"].([]any)) != 2 {
		t.Fatal(m)
	}
}

func TestLang(t *testing.T) {
	en := map[string]string{}
	b, _ := langFS.ReadFile("lang/en.jsonc")
	if err := json.Unmarshal(jsonc(b), &en); err != nil {
		t.Fatal("en.jsonc:", err)
	}
	fs, _ := langFS.ReadDir("lang")
	for _, f := range fs {
		n, m := f.Name(), map[string]string{}
		b, _ := langFS.ReadFile("lang/" + n)
		if err := json.Unmarshal(jsonc(b), &m); err != nil {
			t.Errorf("%s: %v", n, err)
			continue
		}
		for k, v := range en {
			if s, ok := m[k]; !ok {
				t.Errorf("%s: missing %s", n, k)
			} else if verbs(s) != verbs(v) {
				t.Errorf("%s: %s has %d args, en has %d", n, k, verbs(s), verbs(v))
			}
		}
		for k := range m {
			if _, ok := en[k]; !ok {
				t.Errorf("%s: %s not in en.jsonc", n, k)
			}
		}
	}
}
