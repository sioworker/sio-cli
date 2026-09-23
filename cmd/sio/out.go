package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"

	"sio-cli"
)

const red, grn, ylw, cyn, gry, dim, bold = "31", "32", "33", "36", "90", "2", "1"

var colOut, colErr = tty(os.Stdout), tty(os.Stderr)

func tty(f *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func col(on bool, c, s string) string {
	if !on {
		return s
	}
	return "\x1b[" + c + "m" + s + "\x1b[0m"
}

func link(on bool, url, s string) string { // osc 8, plain url when piped
	if !on {
		return s + " " + url
	}
	return "\x1b]8;;" + url + "\x1b\\" + s + "\x1b]8;;\x1b\\"
}

func co(c, s string) string { return col(colOut, c, s) }
func ce(c, s string) string { return col(colErr, c, s) }

func ok(s string)   { fmt.Println(co(grn, "✓"), s) }
func warn(s string) { fmt.Fprintln(os.Stderr, ce(ylw, "!"), s) }

func die(s string) {
	fmt.Fprintln(os.Stderr, ce(red, "✗"), s)
	os.Exit(1)
}

func box(t string, rows [][2]string) {
	n := utf8.RuneCountInString
	kw, w := 0, n(t)+2
	for _, r := range rows {
		kw = max(kw, n(r[0]))
	}
	for _, r := range rows {
		w = max(w, kw+2+n(r[1]))
	}
	fmt.Println(co(dim, "╭─ ") + co(bold+";"+grn, t) + " " + co(dim, strings.Repeat("─", w-n(t)-1)+"╮"))
	for _, r := range rows {
		fmt.Println(co(dim, "│ ") + co(dim, r[0]+strings.Repeat(" ", kw-n(r[0]))) + "  " + co(cyn, r[1]) + strings.Repeat(" ", w-kw-2-n(r[1])) + co(dim, " │"))
	}
	fmt.Println(co(dim, "╰"+strings.Repeat("─", w+2)+"╯"))
}

type cell struct{ s, c string }

func tbl(out io.Writer, hd []string, rows [][]cell) {
	n := utf8.RuneCountInString
	w := make([]int, len(hd))
	for i, h := range hd {
		w[i] = n(h)
	}
	for _, r := range rows {
		for i, c := range r {
			w[i] = max(w[i], n(c.s))
		}
	}
	ln := func(r []cell) {
		o := ""
		for i, c := range r {
			s := c.s
			if i < len(r)-1 {
				s += strings.Repeat(" ", w[i]-n(s)+2)
			}
			o += co(c.c, s)
		}
		fmt.Fprintln(out, o)
	}
	if colOut { // no header when piped
		h := []cell{}
		for _, s := range hd {
			h = append(h, cell{s, dim + ";" + bold})
		}
		ln(h)
	}
	for _, r := range rows {
		ln(r)
	}
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m|\x1b\]8;;[^\x1b]*\x1b\\\\`)

func rows(s string, w int) int { // screen rows incl wrapping
	n := 0
	for _, l := range strings.Split(strings.TrimSuffix(s, "\n"), "\n") {
		n += max(1, (utf8.RuneCountInString(ansiRe.ReplaceAllString(l, ""))+w-1)/w)
	}
	return n
}

func page(hd, s string) { // hd shown first, then all of it in less if taller than the term
	all := s
	if hd != "" {
		fmt.Println(hd)
		all = hd + "\n" + s
	}
	fd := int(os.Stdout.Fd())
	w, ht, err := term.GetSize(fd)
	if !term.IsTerminal(fd) || err != nil || w < 1 || rows(all, w) < ht { // < so the prompt still fits
		fmt.Print(s)
		return
	}
	warn(sio.T("too_long"))
	time.Sleep(time.Second)
	if !less(all) {
		fmt.Print(s)
	}
}

func less(s string) bool {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	cmd := exec.Command("less", "-R")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = strings.NewReader(s), os.Stdout, os.Stderr
	return cmd.Run() == nil
}

func score(v any) string { // api gives int, null or raw "int:000100"
	switch x := v.(type) {
	case float64:
		return fmt.Sprint(int(x))
	case string:
		if _, s, ok := strings.Cut(x, ":"); ok {
			if s = strings.TrimLeft(s, "0"); s == "" {
				s = "0"
			}
			return s
		}
		if x != "" {
			return x
		}
	}
	return "-"
}

func stat(s string) cell {
	switch s {
	case "OK", "INI_OK":
		return cell{"✓ " + s, grn}
	case "", "?":
		return cell{"⋯ " + sio.T("pending"), ylw}
	}
	return cell{"✗ " + s, red}
}
