package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sio-cli"
)

func guess(arg string) (string, string, error) { // -> prob, file
	if ext := filepath.Ext(arg); ext != "" {
		return strings.TrimSuffix(filepath.Base(arg), ext), arg, nil
	}
	if _, err := os.Stat(arg + ".cpp"); err == nil {
		return arg, arg + ".cpp", nil
	}
	m, _ := filepath.Glob(arg + ".*")
	if len(m) == 1 {
		return filepath.Base(arg), m[0], nil
	}
	if len(m) > 1 {
		return arg, "", errors.New(sio.T("multi_file", arg+".*"))
	}
	return arg, "", errors.New(sio.T("no_file", arg))
}

func resolve(arg string) (string, string) {
	p, f, err := guess(arg)
	if err != nil {
		die(err.Error())
	}
	return p, f
}

const noAPI = "!" // Sub.Status when the host cant list subs at all

const noSess = "!s" // same but logged in and the session died

func judge(h *sio.Host, ct, p, id string, tick func(time.Duration)) (sio.Sub, bool) { // poll till id isnt pending, false = gave up
	t0, dl, web := time.Now(), 1500*time.Millisecond, false
	for time.Since(t0) < 5*time.Minute {
		time.Sleep(dl)
		dl = min(dl+500*time.Millisecond, 4*time.Second)
		tick(time.Since(t0))
		if web { // old oioioi, read the my submissions page w/ the login cookie
			s, err := h.SubWeb(ct, id)
			if errors.Is(err, sio.ErrSess) {
				return sio.Sub{Status: noSess}, true
			}
			if err == nil && s.Status != "?" {
				return s, true
			}
			continue
		}
		ss, _, err := h.Subs(ct, p)
		if err != nil && (strings.HasPrefix(err.Error(), "404") || errors.Is(err, sio.ErrHTML)) { // old oioioi (camp) has no problem_submission_list
			if h.Session == "" {
				return sio.Sub{Status: noAPI}, true // waiting wont help, needs sio config hosts login
			}
			web, dl = true, 0
			continue
		}
		if err != nil {
			continue // flaky net, keep trying
		}
		for _, s := range ss {
			if fmt.Sprint(s.ID) == id && s.Status != "?" { // "" = hidden by the contest
				return s, true
			}
		}
	}
	return sio.Sub{Status: "?"}, false
}

func verdict(s sio.Sub) (string, string) { // -> "✓ OK 100", color
	st := stat(s.Status)
	if s.Status == noAPI || s.Status == noSess {
		return sio.T("no_judge"), ylw
	}
	if s.Status == "" {
		return sio.T("hidden"), dim
	}
	if s.Score != nil {
		st.s += " " + fmt.Sprint(*s.Score)
		st.c = scol(*s.Score)
	}
	return st.s, st.c
}

func upCmd(c *sio.Cfg, args []string) {
	nw, rest := false, []string{}
	for _, a := range args {
		if a == "-n" || a == "--no-wait" {
			nw = true
		} else {
			rest = append(rest, a)
		}
	}
	args = rest
	need(args, 2, "submit [-n] <[host/]contest> <prob|file> [file]")
	hn, h, ct := hct(c, args[0])
	prob, file := args[1], ""
	if len(args) > 2 {
		file = args[2]
	} else {
		prob, file = resolve(args[1])
	}
	var id string
	var err error
	wait(sio.T("w_up", file, hn+"/"+ct+"/"+prob), func() { id, err = h.Submit(ct, prob, file) })
	if err != nil {
		fail(hn, err)
	}
	rows := [][2]string{{sio.T("k_file"), file}, {sio.T("k_prob"), hn + "/" + ct + "/" + prob}, {sio.T("k_id"), id}, {sio.T("k_url"), h.URL + "/c/" + ct + "/s/" + id + "/"}}
	if nw {
		box("✓ "+sio.T("sub_ok"), grn, rows)
		return
	}
	var s sio.Sub
	var done bool
	wait(sio.T("w_judge", prob, "0s"), func() {
		s, done = judge(h, ct, prob, id, func(d time.Duration) { spinSet(sio.T("w_judge", prob, d.Round(time.Second).String())) })
	})
	v, vc := verdict(s)
	rows = append(rows[:3], [2]string{sio.T("k_stat"), v}, rows[3])
	box(v, vc, rows)
	u := h.URL + "/c/" + ct + "/s/" + id + "/"
	if s.Status == noAPI {
		warn(sio.T("no_judge_why", ce(cyn, hn)) + " " + sio.T("need_login", ce(cyn, "sio config hosts login "+hn)))
		fmt.Fprintln(os.Stderr, "  "+ce(gry, link(colErr, u, sio.T("open_web"))))
	}
	if s.Status == noSess {
		warn(sio.T("sess_exp", ce(cyn, hn), ce(cyn, "sio config hosts login "+hn)))
		fmt.Fprintln(os.Stderr, "  "+ce(gry, link(colErr, u, sio.T("open_web"))))
	}
	if !done {
		warn(sio.T("judge_slow", ce(cyn, "sio subs "+hn+"/"+ct+" "+prob)))
	}
}
