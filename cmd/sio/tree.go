package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/term"

	"sio-cli"
)

func treeCmd(c *sio.Cfg, args []string) {
	n := ""
	if len(args) > 0 {
		n = args[0]
	}
	n, h := host(c, n)
	na := func() { die(sio.T("tree_na", ce(cyn, n))) }
	var cs []sio.Contest
	var err error
	wait(sio.T("w_contests", n), func() { cs, err = h.Contests() })
	if err != nil && strings.HasPrefix(err.Error(), "404") {
		na()
	}
	if err != nil {
		fail(n, err)
	}
	if len(cs) == 0 {
		warn(sio.T("no_contests", ce(cyn, n)))
		return
	}
	var first []sio.Prob
	wait(sio.T("w_probs", cs[0].ID), func() { first, err = h.Probs(cs[0].ID) })
	if err != nil && strings.HasPrefix(err.Error(), "404") { // no problem_list on old oioioi
		na()
	}
	if err != nil {
		first = nil
	}
	if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		treeUI(n, h, cs, first)
		return
	}
	pss, errs := make([][]sio.Prob, len(cs)), make([]error, len(cs))
	var dn atomic.Int32
	wait(sio.T("w_probs_n", "0/"+fmt.Sprint(len(cs))), func() {
		var wg sync.WaitGroup
		for i, ct := range cs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				pss[i], errs[i] = fetch(h, ct.ID)
				spinSet(sio.T("w_probs_n", fmt.Sprint(dn.Add(1))+"/"+fmt.Sprint(len(cs))))
			}()
		}
		wg.Wait()
	})
	lk := func(u, s string) string { // clickable in a tty, plain text when piped
		if colOut {
			return link(true, u, s)
		}
		return s
	}
	var b strings.Builder
	for i, ct := range cs {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintln(&b, co(bold+";"+cyn, lk(h.URL+"/c/"+ct.ID+"/", ct.ID)), co(dim, ct.Name))
		ps, w := pss[i], 0
		if errs[i] != nil {
			fmt.Fprintln(&b, co(dim, "└── ")+co(red, "✗ "+errs[i].Error()))
			continue
		}
		if len(ps) == 0 {
			fmt.Fprintln(&b, co(dim, "└── "+sio.T("empty")))
		}
		for _, p := range ps {
			w = max(w, len(p.Short))
		}
		for j, p := range ps {
			br := "├── "
			if j == len(ps)-1 {
				br = "└── "
			}
			fmt.Fprintln(&b, co(dim, br)+lk(h.URL+"/c/"+ct.ID+"/p/"+p.Short+"/", co(cyn, p.Short+strings.Repeat(" ", w-len(p.Short)))+"  "+tex(p.Name)))
		}
	}
	fmt.Print(b.String())
}
