package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"sio-cli"
)

const usage = `  sio add-host [--main] <name> <domain> [token]
  sio rm-host <name>
  sio hosts
  sio main <name>
  sio token <name> [token]
  sio ping [name]
  sio upload <[host/]contest> <prob|file> [file]`

func need(args []string, n int, u string) {
	if len(args) < n {
		die(sio.T("usage") + " sio " + u)
	}
}

func host(c *sio.Cfg, name string) (string, *sio.Host) {
	if name == "" {
		name = c.Main
	}
	if name == "" {
		die(sio.T("no_main", ce(cyn, "sio main <name>")))
	}
	h := c.Hosts[name]
	if h == nil {
		die(sio.T("unk_host", ce(cyn, name)))
	}
	return name, h
}

func askToken(h *sio.Host) string {
	fmt.Fprint(os.Stderr, ce(cyn, "? "), sio.T("tok_ask", ce(cyn, h.URL+"/api/token")))
	s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(s)
}

func resolve(arg string) (string, string) { // -> prob, file
	if ext := filepath.Ext(arg); ext != "" {
		return strings.TrimSuffix(filepath.Base(arg), ext), arg
	}
	if _, err := os.Stat(arg + ".cpp"); err == nil {
		return arg, arg + ".cpp"
	}
	m, _ := filepath.Glob(arg + ".*")
	if len(m) == 1 {
		return filepath.Base(arg), m[0]
	}
	if len(m) > 1 {
		die(sio.T("multi_file", ce(cyn, arg+".*")))
	}
	die(sio.T("no_file", ce(cyn, arg)))
	return "", ""
}

func fail(hn string, err error) {
	if strings.HasPrefix(err.Error(), "401") {
		die(sio.T("bad_tok", ce(cyn, hn), ce(cyn, "sio token "+hn)))
	}
	die(err.Error())
}

func main() {
	sio.LoadLang(sio.LangCode())
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, ce(bold, sio.T("usage"))+"\n"+usage)
		os.Exit(1)
	}
	c := sio.Load()
	cmd, args := args[0], args[1:]
	switch cmd {
	case "add-host":
		mk := false
		rest := []string{}
		for _, a := range args {
			if a == "--main" || a == "-m" {
				mk = true
			} else {
				rest = append(rest, a)
			}
		}
		need(rest, 2, "add-host [--main] <name> <domain> [token]")
		u := strings.TrimSuffix(rest[1], "/")
		if !strings.Contains(u, "://") {
			u = "https://" + u
		}
		h := &sio.Host{URL: u}
		if len(rest) > 2 {
			h.Token = rest[2]
		} else {
			h.Token = askToken(h)
		}
		c.Hosts[rest[0]] = h
		if mk || c.Main == "" {
			c.Main = rest[0]
		}
		if err := c.Save(); err != nil {
			die(err.Error())
		}
		if who, err := h.Ping(); err != nil {
			warn(sio.T("added_noauth", ce(cyn, rest[0]), err))
		} else {
			ok(sio.T("added", co(cyn, rest[0]), co(cyn, who)))
		}
	case "rm-host":
		need(args, 1, "rm-host <name>")
		host(c, args[0])
		delete(c.Hosts, args[0])
		if c.Main == args[0] {
			c.Main = ""
		}
		c.Save()
		ok(sio.T("removed", co(cyn, args[0])))
	case "hosts":
		ks := []string{}
		for k := range c.Hosts {
			ks = append(ks, k)
		}
		if len(ks) == 0 {
			warn(sio.T("no_hosts", ce(cyn, "sio add-host")))
		}
		sort.Strings(ks)
		w := 0
		for _, k := range ks {
			w = max(w, len(k))
		}
		for _, k := range ks {
			m := " "
			if k == c.Main {
				m = "*"
			}
			if colOut { // plain markers when piped, completions cut on bytes
				m = co(dim, "○")
				if k == c.Main {
					m = co(grn, "●")
				}
			}
			fmt.Println(m, co(cyn, k+strings.Repeat(" ", w-len(k))), co(dim, c.Hosts[k].URL))
		}
	case "main":
		need(args, 1, "main <name>")
		host(c, args[0])
		c.Main = args[0]
		c.Save()
		ok(sio.T("main_set", co(cyn, args[0])))
	case "token":
		need(args, 1, "token <name> [token]")
		_, h := host(c, args[0])
		if len(args) > 1 {
			h.Token = args[1]
		} else {
			h.Token = askToken(h)
		}
		c.Save()
		ok(sio.T("tok_set", co(cyn, args[0])))
	case "ping":
		n := ""
		if len(args) > 0 {
			n = args[0]
		}
		n, h := host(c, n)
		who, err := h.Ping()
		if err != nil {
			fail(n, err)
		}
		ok(sio.T("ping_ok", co(cyn, n), co(cyn, who)))
	case "upload", "up":
		need(args, 2, "upload <[host/]contest> <prob|file> [file]")
		hn, ct, ok := strings.Cut(args[0], "/")
		if !ok {
			hn, ct = "", hn
		}
		hn, h := host(c, hn)
		prob, file := args[1], ""
		if len(args) > 2 {
			file = args[2]
		} else {
			prob, file = resolve(args[1])
		}
		id, err := h.Submit(ct, prob, file)
		if err != nil {
			fail(hn, err)
		}
		box("✓ "+sio.T("sub_ok"), [][2]string{{sio.T("k_file"), file}, {sio.T("k_prob"), hn + "/" + ct + "/" + prob}, {sio.T("k_id"), id}, {sio.T("k_url"), h.URL + "/c/" + ct + "/s/" + id + "/"}})
	default:
		die(sio.T("unk_cmd", ce(cyn, cmd)) + "\n" + usage)
	}
}
