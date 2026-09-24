package main

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/term"
)

var (
	errMu   sync.Mutex // spinner + warn/die share stderr
	spinMsg atomic.Pointer[string]
	unspin  = func() {}
)

func wait(msg string, f func()) { // spinner on stderr while f runs, tty only
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		f()
		return
	}
	spinMsg.Store(&msg)
	done, wg, once := make(chan bool), sync.WaitGroup{}, sync.Once{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		fr, i, t := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, 0, time.NewTicker(80*time.Millisecond)
		defer t.Stop()
		for {
			errMu.Lock()
			fmt.Fprint(os.Stderr, "\r\x1b[K"+ce(cyn, fr[i%len(fr)])+" "+ce(dim, *spinMsg.Load()))
			errMu.Unlock()
			i++
			select {
			case <-done:
				errMu.Lock()
				fmt.Fprint(os.Stderr, "\r\x1b[K")
				errMu.Unlock()
				return
			case <-t.C:
			}
		}
	}()
	unspin = func() { once.Do(func() { close(done); wg.Wait() }) }
	defer func() { unspin(); unspin = func() {} }()
	f()
}

func spinSet(s string) { spinMsg.Store(&s) }
