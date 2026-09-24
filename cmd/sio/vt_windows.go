//go:build windows

package main

import "golang.org/x/sys/windows"

func init() { // colors, spinner and tree need vt mode, old conhost has it off
	for _, h := range []windows.Handle{windows.Stdout, windows.Stderr} {
		var m uint32
		if windows.GetConsoleMode(h, &m) == nil {
			windows.SetConsoleMode(h, m|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
		}
	}
}
