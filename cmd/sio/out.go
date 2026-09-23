package main

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

const red, grn, ylw, cyn, dim, bold = "31", "32", "33", "36", "2", "1"

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
