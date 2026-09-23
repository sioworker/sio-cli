package main

import (
	"bufio"
	"errors"
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
  sio info [name]
  sio upload <[host/]contest> <prob|file> [file]
  sio probs <[host/]contest>
  sio subs <[host/]contest> [prob]`

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

func hct(c *sio.Cfg, arg string) (string, *sio.Host, string) { // [host/]contest -> host name, host, contest
	hn, ct, ok := strings.Cut(arg, "/")
	if !ok {
		hn, ct = "", hn
	}
	hn, h := host(c, hn)
	return hn, h, ct
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
	if errors.Is(err, sio.ErrHTML) {
		die(sio.T("not_api", ce(cyn, hn)))
	}
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
			if errors.Is(err, sio.ErrHTML) {
				err = errors.New(sio.T("not_api", rest[0]))
			}
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
	case "info":
		n := ""
		if len(args) > 0 {
			n = args[0]
		}
		n, h := host(c, n)
		who, err := h.Ping()
		if err != nil {
			fail(n, err)
		}
		hd := co(grn, "✓") + " " + sio.T("ping_ok", co(cyn, n), co(cyn, who)) + " " + co(dim, h.URL)
		cs, err := h.Contests()
		if err != nil {
			fmt.Println(hd)
		}
		if err != nil && strings.HasPrefix(err.Error(), "404") { // old oioioi, no contest_list
			die(err.Error() + "\n  " + ce(gry, link(colErr, "https://pastebin.com/chMT18MG", sio.T("why"))))
		}
		if err != nil {
			fail(n, err)
		}
		if len(cs) == 0 {
			warn(sio.T("no_contests", ce(cyn, n)))
		}
		rows := [][]cell{}
		for _, ct := range cs {
			rows = append(rows, []cell{{ct.ID, cyn}, {ct.Name, ""}})
		}
		var b strings.Builder
		tbl(&b, []string{sio.T("k_contest"), sio.T("k_name")}, rows)
		page(hd, b.String())
	case "upload", "up":
		need(args, 2, "upload <[host/]contest> <prob|file> [file]")
		hn, h, ct := hct(c, args[0])
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
	case "probs":
		need(args, 1, "probs <[host/]contest>")
		hn, h, ct := hct(c, args[0])
		ps, err := h.Probs(ct)
		if err != nil {
			fail(hn, err)
		}
		if len(ps) == 0 {
			warn(sio.T("no_probs", ce(cyn, hn+"/"+ct)))
		}
		rows := [][]cell{}
		for _, p := range ps {
			st, sc, l := cell{"", ""}, "-", "-"
			if p.Res != nil && p.Res.Status != "" {
				st, sc = stat(p.Res.Status), score(p.Res.Score)
			}
			if p.Left != nil {
				l = fmt.Sprint(*p.Left)
			}
			rows = append(rows, []cell{{p.Short, cyn}, {p.Name, ""}, {sc, bold}, {l, dim}, st})
		}
		tbl(os.Stdout, []string{sio.T("k_prob"), sio.T("k_name"), sio.T("k_score"), sio.T("k_left"), sio.T("k_stat")}, rows)
	case "subs":
		need(args, 1, "subs <[host/]contest> [prob]")
		hn, h, ct := hct(c, args[0])
		pn := []string{}
		if len(args) > 1 {
			pn = args[1:]
		} else {
			ps, err := h.Probs(ct)
			if err != nil {
				fail(hn, err)
			}
			for _, p := range ps {
				pn = append(pn, p.Short)
			}
		}
		ss := []sio.Sub{}
		for _, p := range pn {
			s, tr, err := h.Subs(ct, p)
			if err != nil {
				fail(hn, err)
			}
			if tr {
				warn(sio.T("trunc", ce(cyn, p)))
			}
			ss = append(ss, s...)
		}
		if len(ss) == 0 {
			warn(sio.T("no_subs", ce(cyn, hn+"/"+ct)))
		}
		sort.Slice(ss, func(i, j int) bool { return ss[i].Date.After(ss[j].Date) })
		rows := [][]cell{}
		for _, s := range ss {
			sc := "-"
			if s.Score != nil {
				sc = fmt.Sprint(*s.Score)
			}
			rows = append(rows, []cell{{fmt.Sprint(s.ID), dim}, {s.Prob, cyn}, {s.Date.Local().Format("2006-01-02 15:04"), ""}, {sc, bold}, stat(s.Status)})
		}
		tbl(os.Stdout, []string{sio.T("k_id"), sio.T("k_prob"), sio.T("k_date"), sio.T("k_score"), sio.T("k_stat")}, rows)
	default:
		die(sio.T("unk_cmd", ce(cyn, cmd)) + "\n" + usage)
	}
}
