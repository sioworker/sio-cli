package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

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

func upCmd(c *sio.Cfg, args []string) {
	need(args, 2, "upload <[host/]contest> <prob|file> [file]")
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
	box("✓ "+sio.T("sub_ok"), [][2]string{{sio.T("k_file"), file}, {sio.T("k_prob"), hn + "/" + ct + "/" + prob}, {sio.T("k_id"), id}, {sio.T("k_url"), h.URL + "/c/" + ct + "/s/" + id + "/"}})
}
