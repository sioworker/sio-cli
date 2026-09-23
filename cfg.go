package sio

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Host struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type Cfg struct {
	Main  string           `json:"main"`
	Lang  string           `json:"lang,omitempty"`
	Hosts map[string]*Host `json:"hosts"`
}

func CfgPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "sio", "cfg.json")
}

func Load() *Cfg {
	c := &Cfg{Hosts: map[string]*Host{}}
	b, err := os.ReadFile(CfgPath())
	if err == nil {
		json.Unmarshal(b, c)
	}
	if c.Hosts == nil {
		c.Hosts = map[string]*Host{}
	}
	return c
}

func (c *Cfg) Save() error {
	p := CfgPath()
	os.MkdirAll(filepath.Dir(p), 0700)
	b, _ := json.MarshalIndent(c, "", "\t")
	return os.WriteFile(p, b, 0600) // has tokens
}
