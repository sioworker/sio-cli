package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"sio-cli"
)

func onoff(b bool) string {
	if b {
		return sio.T("on")
	}
	return sio.T("off")
}

func cfgCmd(c *sio.Cfg, args []string) {
	if len(args) == 0 {
		src, lc := sio.T("src_auto"), sio.LangCode(c.Lang)
		if os.Getenv("SIO_LANG") != "" {
			src = sio.T("src_env")
		} else if c.Lang != "" {
			src = sio.T("src_saved")
		}
		mn := c.Main
		if mn == "" {
			mn = "-"
		}
		rows := [][]cell{{{sio.T("k_cfg"), dim}, {sio.CfgPath(), ""}}, {{sio.T("k_hosts"), dim}, {sio.T("hosts_sum", fmt.Sprint(len(c.Hosts)), mn), ""}}, {{sio.T("k_lang"), dim}, {lc + " (" + sio.LangName(lc) + "), " + src, ""}}, {{sio.T("k_quotes"), dim}, {onoff(c.QuotesOn()), ""}}}
		tbl(os.Stdout, []string{"", ""}, rows)
		fmt.Fprintln(os.Stderr, ce(dim, cfgUsage))
		return
	}
	cat, args := args[0], args[1:]
	switch cat {
	case "hosts":
		if len(args) == 0 {
			ks := []string{}
			for k := range c.Hosts {
				ks = append(ks, k)
			}
			if len(ks) == 0 {
				warn(sio.T("no_hosts", ce(cyn, "sio config hosts add")))
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
			return
		}
		sub, args := args[0], args[1:]
		switch sub {
		case "add":
			mk := false
			rest := []string{}
			for _, a := range args {
				if a == "--main" || a == "-m" {
					mk = true
				} else {
					rest = append(rest, a)
				}
			}
			need(rest, 2, "config hosts add [--main] <name> <domain> [token]")
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
		case "rm":
			need(args, 1, "config hosts rm <name>")
			host(c, args[0])
			delete(c.Hosts, args[0])
			if c.Main == args[0] {
				c.Main = ""
			}
			c.Save()
			ok(sio.T("removed", co(cyn, args[0])))
		case "main":
			need(args, 1, "config hosts main <name>")
			host(c, args[0])
			c.Main = args[0]
			c.Save()
			ok(sio.T("main_set", co(cyn, args[0])))
		case "token":
			need(args, 1, "config hosts token <name> [token]")
			_, h := host(c, args[0])
			if len(args) > 1 {
				h.Token = args[1]
			} else {
				h.Token = askToken(h)
			}
			c.Save()
			ok(sio.T("tok_set", co(cyn, args[0])))
		default:
			die(sio.T("unk_cmd", ce(cyn, "config hosts "+sub)) + "\n" + cfgUsage)
		}
	case "quotes":
		if len(args) == 0 {
			fmt.Println(sio.T("k_quotes")+":", co(cyn, onoff(c.QuotesOn())))
			return
		}
		if args[0] != "on" && args[0] != "off" {
			die(sio.T("usage") + " sio config quotes on|off")
		}
		v := args[0] == "on"
		c.Quotes = &v
		if err := c.Save(); err != nil {
			die(err.Error())
		}
		ok(sio.T("quotes_set", co(cyn, onoff(v))))
	case "lang":
		ls := sio.Langs()
		if len(args) == 0 {
			cur, w := sio.LangCode(c.Lang), 0
			for _, l := range ls {
				w = max(w, len(l))
			}
			for _, l := range ls {
				m := " "
				if l == cur {
					m = "*"
				}
				if colOut {
					m = co(dim, "○")
					if l == cur {
						m = co(grn, "●")
					}
				}
				fmt.Println(m, co(cyn, l+strings.Repeat(" ", w-len(l))), sio.LangName(l))
			}
			if v := os.Getenv("SIO_LANG"); v != "" {
				warn(sio.T("lang_env", ce(cyn, "SIO_LANG="+v)))
			}
			return
		}
		if args[0] == "auto" {
			c.Lang = ""
		} else if !slices.Contains(ls, args[0]) {
			die(sio.T("unk_lang", ce(cyn, args[0]), ce(cyn, "sio config lang")))
		} else {
			c.Lang = args[0]
		}
		if err := c.Save(); err != nil {
			die(err.Error())
		}
		sio.L = map[string]string{}
		sio.LoadLang(sio.LangCode(c.Lang)) // confirm in the new lang
		ok(sio.T("lang_set", co(cyn, sio.LangCode(c.Lang)), sio.LangName(sio.LangCode(c.Lang))))
		if v := os.Getenv("SIO_LANG"); v != "" {
			warn(sio.T("lang_env", ce(cyn, "SIO_LANG="+v)))
		}
	default:
		die(sio.T("unk_cmd", ce(cyn, "config "+cat)) + "\n" + cfgUsage)
	}
}
