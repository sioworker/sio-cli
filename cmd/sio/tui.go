package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/term"

	"sio-cli"
)

type tnode struct {
	ct         sio.Contest
	open, busy bool
	ps         []sio.Prob
	err        error
}

type trow struct{ c, p int } // p -1 = contest, -2 = status line

type seg struct{ s, c string }

func fit(ss []seg, w int) string { // cut to w cols, colors kept
	o := ""
	for _, g := range ss {
		if w <= 0 {
			break
		}
		if n := utf8.RuneCountInString(g.s); n > w {
			g.s = string([]rune(g.s)[:w])
		}
		w -= utf8.RuneCountInString(g.s)
		o += co(g.c, g.s)
	}
	return o
}

func browse(u string) {
	c, a := "xdg-open", []string{u}
	switch runtime.GOOS {
	case "darwin":
		c = "open"
	case "windows":
		c, a = "rundll32", []string{"url.dll,FileProtocolHandler", u}
	}
	cmd := exec.Command(c, a...)
	if cmd.Start() == nil {
		go cmd.Wait()
	}
}

func fetch(h *sio.Host, ct string) ([]sio.Prob, error) {
	ps, err := h.Probs(ct)
	if err != nil && strings.HasPrefix(err.Error(), "5") { // same web fallback as probs
		ps, err = h.ProbsWeb(ct)
	}
	return ps, err
}

func treeUI(hn string, h *sio.Host, cs []sio.Contest, first []sio.Prob) {
	fd := int(os.Stdin.Fd())
	st, err := term.MakeRaw(fd)
	if err != nil {
		die(err.Error())
	}
	defer term.Restore(fd, st)
	fmt.Print("\x1b[?1049h\x1b[?25l") // alt screen, hide cursor
	defer fmt.Print("\x1b[?25h\x1b[?1049l")
	var mu sync.Mutex
	ns := make([]*tnode, len(cs))
	for i, ct := range cs {
		ns[i] = &tnode{ct: ct}
	}
	ns[0].ps = first
	redraw, keys := make(chan bool, 1), make(chan string)
	go func() {
		b := make([]byte, 32)
		for {
			n, err := os.Stdin.Read(b)
			if err != nil {
				close(keys)
				return
			}
			keys <- string(b[:n])
		}
	}()
	tick := time.NewTicker(100 * time.Millisecond) // catch resizes (no SIGWINCH on windows) + spin
	defer tick.Stop()
	rows := func() []trow {
		rs := []trow{}
		for i, n := range ns {
			rs = append(rs, trow{i, -1})
			if !n.open {
				continue
			}
			if n.busy || n.err != nil || len(n.ps) == 0 {
				rs = append(rs, trow{i, -2})
			}
			for j := range n.ps {
				rs = append(rs, trow{i, j})
			}
		}
		return rs
	}
	cur, top, lw, lh, rev := 0, 0, 0, 0, -1 // rev = contest to scroll into view once opened
	type inp struct {
		c, p int
		buf  []rune
	}
	var in *inp // file prompt after u
	busy, upMsg, note := false, "", seg{}
	draw := func() {
		mu.Lock()
		defer mu.Unlock()
		w, ht, _ := term.GetSize(int(os.Stdout.Fd()))
		lw, lh = w, ht
		rs, bh := rows(), max(1, ht-2)
		cur = min(max(cur, 0), len(rs)-1)
		if cur < top {
			top = cur
		}
		if cur >= top+bh {
			top = cur - bh + 1
		}
		if rev >= 0 {
			last := cur
			for last+1 < len(rs) && rs[last+1].c == rev {
				last++
			}
			if last >= top+bh {
				top = min(cur, last-bh+1)
			}
			if !ns[rev].busy {
				rev = -1
			}
		}
		var b strings.Builder
		b.WriteString("\x1b[H" + fit([]seg{{"✓ ", grn}, {hn, bold + ";" + cyn}, {" " + h.URL, dim}}, w) + "\x1b[K\r\n")
		i := top
		for i < top+bh {
			if i < len(rs) {
				r, mk := rs[i], seg{"  ", ""}
				if i == cur {
					mk = seg{"❯ ", bold + ";" + cyn}
				}
				n := ns[r.c]
				switch {
				case r.p == -1:
					ar := "▸ "
					if n.open {
						ar = "▾ "
					}
					cst := cyn
					if i == cur {
						cst = bold + ";" + cyn
					}
					b.WriteString(fit([]seg{mk, {ar, dim}, {n.ct.ID, cst}, {" " + n.ct.Name, dim}}, w))
				case r.p == -2:
					s := seg{sio.T("empty"), dim}
					if n.busy {
						s = seg{"⋯ " + sio.T("loading"), ylw}
					} else if n.err != nil {
						s = seg{"✗ " + n.err.Error(), red}
					}
					b.WriteString(fit([]seg{mk, {"  └── ", dim}, s}, w))
				default:
					p, pw := n.ps[r.p], 0
					for _, q := range n.ps {
						pw = max(pw, len(q.Short))
					}
					br := "  ├── "
					if r.p == len(n.ps)-1 {
						br = "  └── "
					}
					nm := seg{tex(p.Name), ""}
					if i == cur {
						nm.c = bold
					}
					b.WriteString(link(colOut, h.URL+"/c/"+n.ct.ID+"/p/"+p.Short+"/", fit([]seg{mk, {br, dim}, {p.Short + strings.Repeat(" ", pw-len(p.Short)), cyn}, {"  ", ""}, nm}, w)))
				}
			}
			b.WriteString("\x1b[K\r\n")
			i++
		}
		ft := []seg{{sio.T("tree_help"), dim}}
		switch {
		case in != nil:
			n := ns[in.c]
			ft = []seg{{"↑ " + sio.T("up_ask", n.ct.ID+"/"+n.ps[in.p].Short), cyn}, {string(in.buf), ""}, {"█", dim}}
		case busy:
			fr := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			ft = []seg{{fr[time.Now().UnixMilli()/100%10] + " ", cyn}, {upMsg, dim}}
			if note.s != "" {
				ft = append(ft, seg{"  " + note.s, note.c})
			}
		case note.s != "":
			ft = []seg{note}
		}
		b.WriteString(fit(ft, w) + "\x1b[K\x1b[J")
		fmt.Print(b.String())
	}
	send := func() { // upload in.buf in the bg, mu held
		n, f := ns[in.c], strings.TrimSpace(string(in.buf))
		ct, p := n.ct.ID, n.ps[in.p].Short
		in = nil
		if f == "" {
			note = seg{"✗ " + sio.T("no_file", p), red}
			return
		}
		busy, upMsg = true, sio.T("w_up", f, ct+"/"+p)
		go func() {
			id, err := h.Submit(ct, p, f)
			if err == nil { // wait for the verdict, like sio up
				mu.Lock()
				upMsg = sio.T("w_judge", p, "0s")
				mu.Unlock()
				s, _ := judge(h, ct, p, id, func(d time.Duration) {
					mu.Lock()
					upMsg = sio.T("w_judge", p, d.Round(time.Second).String())
					mu.Unlock()
				})
				v, vc := verdict(s)
				if s.Status == noAPI {
					v += ", " + sio.T("need_login", "sio config hosts login "+hn)
				} else if s.Status == noSess {
					v += ", " + sio.T("sess_exp", hn, "sio config hosts login "+hn)
				}
				mu.Lock()
				busy, note = false, seg{v + "  " + sio.T("up_ok", f, ct+"/"+p, id), vc}
				mu.Unlock()
			} else {
				mu.Lock()
				busy, note = false, seg{"✗ " + err.Error(), red}
				mu.Unlock()
			}
			select {
			case redraw <- true:
			default:
			}
		}()
	}
	open := func(i int) {
		n := ns[i]
		n.open, rev = true, i
		if n.ps != nil || n.busy || n.err != nil {
			return
		}
		n.busy = true
		go func() {
			ps, err := fetch(h, n.ct.ID)
			mu.Lock()
			n.ps, n.err, n.busy = ps, err, false
			mu.Unlock()
			select {
			case redraw <- true:
			default:
			}
		}()
	}
	for {
		draw()
		select {
		case <-redraw:
		case <-tick.C:
			mu.Lock()
			b := busy
			mu.Unlock()
			if w, ht, _ := term.GetSize(int(os.Stdout.Fd())); w == lw && ht == lh && !b {
				continue
			}
		case k, ok := <-keys:
			if !ok {
				return
			}
			mu.Lock()
			if in != nil { // typing the file path
				if k == "\x03" {
					mu.Unlock()
					return
				}
				if k == "\x1b" {
					in = nil
				} else if !strings.HasPrefix(k, "\x1b") { // arrows etc ignored
					for _, ch := range k { // a chunk can hold a few keys or a paste
						if ch == '\r' || ch == '\n' {
							send()
							break
						} else if ch == 127 || ch == 8 {
							if len(in.buf) > 0 {
								in.buf = in.buf[:len(in.buf)-1]
							}
						} else if ch >= 32 {
							in.buf = append(in.buf, ch)
						}
					}
				}
				mu.Unlock()
				continue
			}
			rs, bh := rows(), max(1, lh-2)
			r := rs[min(max(cur, 0), len(rs)-1)]
			if !busy {
				note = seg{}
			}
			switch k {
			case "\x03":
				mu.Unlock()
				return
			case "q", "\x1b":
				if busy { // quitting would kill the upload mid way
					note = seg{"! " + sio.T("up_busy"), ylw}
					break
				}
				mu.Unlock()
				return
			case "u":
				if busy {
					note = seg{"! " + sio.T("up_busy"), ylw}
				} else if r.p >= 0 {
					_, f, _ := guess(ns[r.c].ps[r.p].Short)
					in = &inp{r.c, r.p, []rune(f)}
				} else {
					note = seg{"! " + sio.T("up_pick"), ylw}
				}
			case "\x1b[A", "k":
				cur--
			case "\x1b[B", "j":
				cur++
			case "\x1b[5~":
				cur -= bh
			case "\x1b[6~":
				cur += bh
			case "\x1b[H", "\x1b[1~", "g":
				cur = 0
			case "\x1b[F", "\x1b[4~", "G":
				cur = len(rs) - 1
			case "\x1b[C", "l":
				if r.p == -1 {
					open(r.c)
				}
			case "\x1b[D", "h":
				if r.p == -1 {
					ns[r.c].open = false
				} else {
					for cur > 0 && rs[cur].p != -1 {
						cur--
					}
				}
			case " ", "\r", "\n":
				if r.p == -1 {
					if ns[r.c].open {
						ns[r.c].open = false
					} else {
						open(r.c)
					}
				} else if r.p >= 0 {
					browse(h.URL + "/c/" + ns[r.c].ct.ID + "/p/" + ns[r.c].ps[r.p].Short + "/")
				}
			}
			mu.Unlock()
		}
	}
}
