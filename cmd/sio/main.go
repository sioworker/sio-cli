package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"sio-cli"
)

const cfgUsage = `  sio config [hosts|lang]
  sio config hosts [add|rm|main|token]
  sio config hosts add [--main] <name> <domain> [token]
  sio config hosts rm <name>
  sio config hosts main <name>
  sio config hosts token <name> [token]
  sio config lang [code|auto]`

const usage = cfgUsage + `
  sio ping [name]
  sio info [name]
  sio tree [name]
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
		die(sio.T("no_main", ce(cyn, "sio config hosts main <name>")))
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

func fail(hn string, err error) {
	if errors.Is(err, sio.ErrHTML) {
		die(sio.T("not_api", ce(cyn, hn)))
	}
	if strings.HasPrefix(err.Error(), "401") {
		die(sio.T("bad_tok", ce(cyn, hn), ce(cyn, "sio config hosts token "+hn)))
	}
	die(err.Error())
}

func main() {
	c := sio.Load()
	sio.LoadLang(sio.LangCode(c.Lang))
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, ce(bold, sio.T("usage"))+"\n"+usage)
		os.Exit(1)
	}
	cmd, args := args[0], args[1:]
	switch cmd {
	case "config", "cfg":
		cfgCmd(c, args)
	case "ping":
		pingCmd(c, args)
	case "info":
		infoCmd(c, args)
	case "tree":
		treeCmd(c, args)
	case "upload", "up":
		upCmd(c, args)
	case "probs":
		probsCmd(c, args)
	case "subs":
		subsCmd(c, args)
	default:
		die(sio.T("unk_cmd", ce(cyn, cmd)) + "\n" + usage)
	}
}
