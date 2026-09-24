package main

import (
	"fmt"
	"strings"

	"sio-cli"
)

func pingCmd(c *sio.Cfg, args []string) {
	n := ""
	if len(args) > 0 {
		n = args[0]
	}
	n, h := host(c, n)
	var who string
	var err error
	wait(sio.T("w_ping", n), func() { who, err = h.Ping() })
	if err != nil {
		fail(n, err)
	}
	ok(sio.T("ping_ok", co(cyn, n), co(cyn, who)))
}

func infoCmd(c *sio.Cfg, args []string) {
	n := ""
	if len(args) > 0 {
		n = args[0]
	}
	n, h := host(c, n)
	var who string
	var cs []sio.Contest
	var err error
	wait(sio.T("w_ping", n), func() { who, err = h.Ping() })
	if err != nil {
		fail(n, err)
	}
	hd := co(grn, "✓") + " " + sio.T("ping_ok", co(cyn, n), co(cyn, who)) + " " + co(dim, h.URL)
	wait(sio.T("w_contests", n), func() { cs, err = h.Contests() })
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
}
