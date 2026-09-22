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

const usage = `usage:
  sio add-host [--main] <name> <domain> [token]
  sio rm-host <name>
  sio hosts
  sio main <name>
  sio token <name> [token]
  sio ping [name]
  sio upload <[host/]contest> <prob|file> [file]`

func die(a ...any) {
	fmt.Fprintln(os.Stderr, append([]any{"sio:"}, a...)...)
	os.Exit(1)
}

func need(args []string, n int, u string) {
	if len(args) < n {
		die("usage: sio", u)
	}
}

func host(c *sio.Cfg, name string) (string, *sio.Host) {
	if name == "" {
		name = c.Main
	}
	if name == "" {
		die("no host given and no main host set (sio main <name>)")
	}
	h := c.Hosts[name]
	if h == nil {
		die("unknown host", name)
	}
	return name, h
}

func askToken(h *sio.Host) string {
	fmt.Fprintf(os.Stderr, "token (get it at %s/api/token): ", h.URL)
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
		die("more than one", arg+".*, pass the file explicitly")
	}
	die("no file for", arg)
	return "", ""
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
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
			die(err)
		}
		if who, err := h.Ping(); err != nil {
			fmt.Fprintln(os.Stderr, "added, but auth failed:", err)
		} else {
			fmt.Println("added", rest[0], "-", who)
		}
	case "rm-host":
		need(args, 1, "rm-host <name>")
		delete(c.Hosts, args[0])
		if c.Main == args[0] {
			c.Main = ""
		}
		c.Save()
	case "hosts":
		ks := []string{}
		for k := range c.Hosts {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		for _, k := range ks {
			m := " "
			if k == c.Main {
				m = "*"
			}
			fmt.Println(m, k, c.Hosts[k].URL)
		}
	case "main":
		need(args, 1, "main <name>")
		host(c, args[0])
		c.Main = args[0]
		c.Save()
	case "token":
		need(args, 1, "token <name> [token]")
		_, h := host(c, args[0])
		if len(args) > 1 {
			h.Token = args[1]
		} else {
			h.Token = askToken(h)
		}
		c.Save()
	case "ping":
		n := ""
		if len(args) > 0 {
			n = args[0]
		}
		n, h := host(c, n)
		who, err := h.Ping()
		if err != nil {
			die(n+":", err)
		}
		fmt.Println(n+":", who)
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
			die(err)
		}
		fmt.Printf("%s -> %s/%s/%s, submission %s\n%s/c/%s/s/%s/\n", file, hn, ct, prob, id, h.URL, ct, id)
	default:
		die("unknown cmd", cmd+"\n"+usage)
	}
}
