package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"sio-cli"
)

func probs(hn string, h *sio.Host, ct string) []sio.Prob {
	var ps []sio.Prob
	var err error
	web := false
	wait(sio.T("w_probs", hn+"/"+ct), func() {
		ps, err = h.Probs(ct)
		if err != nil && strings.HasPrefix(err.Error(), "5") { // problem_list 500s on some contests, fall back to the web page
			ps, err = h.ProbsWeb(ct)
			web = err == nil
		}
	})
	if web {
		warn(sio.T("probs_web", ce(cyn, hn+"/"+ct)))
	}
	if err != nil {
		fail(hn, err)
	}
	return ps
}

var texRe, texCmd = regexp.MustCompile(`\\[a-zA-Z]+\{([^{}]*)\}`), regexp.MustCompile(`\\[a-zA-Z]+ ?`)

func tex(s string) string { // $k$-inwersje, \mbox{x} -> k-inwersje, x
	for texRe.MatchString(s) {
		s = texRe.ReplaceAllString(s, "$1")
	}
	return strings.TrimSpace(texCmd.ReplaceAllString(strings.ReplaceAll(s, "$", ""), ""))
}

func probsCmd(c *sio.Cfg, args []string) {
	need(args, 1, "probs <[host/]contest>")
	hn, h, ct := hct(c, args[0])
	ps := probs(hn, h, ct)
	if len(ps) == 0 {
		warn(sio.T("no_probs", ce(cyn, hn+"/"+ct)))
	}
	rows := [][]cell{}
	for _, p := range ps {
		st, sc, l := cell{"-", dim}, "-", "-"
		if p.Res != nil && p.Res.Status != "" {
			st, sc = stat(p.Res.Status), score(p.Res.Score)
		}
		if p.Left != nil {
			l = fmt.Sprint(*p.Left)
		}
		rows = append(rows, []cell{{p.Short, cyn}, {tex(p.Name), ""}, {sc, bold}, {l, dim}, st})
	}
	var b strings.Builder
	tbl(&b, []string{sio.T("k_prob"), sio.T("k_name"), sio.T("k_score"), sio.T("k_left"), sio.T("k_stat")}, rows)
	page("", b.String())
}

func subsCmd(c *sio.Cfg, args []string) {
	need(args, 1, "subs <[host/]contest> [prob]")
	hn, h, ct := hct(c, args[0])
	pn := []string{}
	if len(args) > 1 {
		pn = args[1:]
	} else {
		for _, p := range probs(hn, h, ct) {
			pn = append(pn, p.Short)
		}
	}
	ss := []sio.Sub{}
	for i, p := range pn {
		var s []sio.Sub
		var tr bool
		var err error
		wait(sio.T("w_subs", p, fmt.Sprint(i+1)+"/"+fmt.Sprint(len(pn))), func() { s, tr, err = h.Subs(ct, p) })
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
	var b strings.Builder
	tbl(&b, []string{sio.T("k_id"), sio.T("k_prob"), sio.T("k_date"), sio.T("k_score"), sio.T("k_stat")}, rows)
	page("", b.String())
}
