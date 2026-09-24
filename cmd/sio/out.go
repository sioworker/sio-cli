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

func ok(s string) { fmt.Println(co(grn, "✓"), s) }
func warn(s string) { // clears a running spinner line first, it redraws below
	errMu.Lock()
	defer errMu.Unlock()
	fmt.Fprint(os.Stderr, "\r\x1b[K")
	fmt.Fprintln(os.Stderr, ce(ylw, "!"), s)
}

func die(s string) {
	unspin()
	fmt.Fprintln(os.Stderr, ce(red, "✗"), s)
	os.Exit(1)
}

func cut(s string, w int) string { // to w cols, … if cut
	if w < 1 {
		return ""
	}
	if r := []rune(s); len(r) > w {
		return string(r[:w-1]) + "…"
	}
	return s
}

func box(t, tc string, rows [][2]string) { // tc = title color, shrinks to the term width
	n := utf8.RuneCountInString
	kw, w := 0, n(t)+2
	for _, r := range rows {
		kw = max(kw, n(r[0]))
	}
	for _, r := range rows {
		w = max(w, kw+2+n(r[1]))
	}
	if tw, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w+4 > tw { // wider than the term would wrap every line
		w = max(tw-4, kw+4)
		t = cut(t, w-2)
	}
	fmt.Println(co(dim, "╭─ ") + co(bold+";"+tc, t) + " " + co(dim, strings.Repeat("─", w-n(t)-1)+"╮"))
	for _, r := range rows {
		vc, v := cyn, cut(r[1], w-kw-2)
		if r[0] == sio.T("k_stat") { // status row takes the verdict color
			vc = tc
		}
		cv := co(vc, v)
		if r[0] == sio.T("k_url") && colOut { // full url stays clickable even when cut
			cv = link(true, r[1], cv)
		}
		fmt.Println(co(dim, "│ ") + co(dim, r[0]+strings.Repeat(" ", kw-n(r[0]))) + "  " + cv + strings.Repeat(" ", w-kw-2-n(v)) + co(dim, " │"))
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
	if colOut && strings.Join(hd, "") != "" { // no header when piped or blank
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

func wrap(s string, w int) []string {
	ls, l := []string{}, ""
	for _, x := range strings.Fields(s) {
		if l != "" && utf8.RuneCountInString(l+" "+x) > w {
			ls, l = append(ls, l), x
		} else if l == "" {
			l = x
		} else {
			l += " " + x
		}
	}
	return append(ls, l)
}

func quote() {
	var q sio.Quote
	var has bool
	if sio.QuotesStale() {
		wait(sio.T("w_quote"), func() { q, has = sio.RandQuote() })
	} else {
		q, has = sio.RandQuote() // cached, instant, no spinner blink
	}
	if !has {
		return
	}
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w < 20 {
		w = 80
	}
	ls := wrap("\""+q.Text+"\"", min(w-4, 76))
	fmt.Println()
	for _, l := range ls {
		fmt.Println("  " + co(dim+";3", l))
	}
	fmt.Println("    " + co(dim, "- ") + co(dim+";"+cyn, q.Author))
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

func scol(n int) string { // <50 red, <80 yellow, else green
	if n < 50 {
		return red
	}
	if n < 80 {
		return ylw
	}
	return grn
}

func scell(s string) cell { // score cell colored by value, "-" stays dim
	var n int
	if _, err := fmt.Sscan(s, &n); err != nil {
		return cell{s, dim}
	}
	return cell{s, bold + ";" + scol(n)}
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
