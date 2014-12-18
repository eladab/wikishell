package main

import (
	"fmt"
	"github.com/pkg/term/termios"
	"syscall"
)

var orig syscall.Termios

const fd = 0

func initTerm() {
	if err := termios.Tcgetattr(uintptr(fd), &orig); err != nil {
		fmt.Println("Failed to get attribute")
		return
	}
}

func SetRawModeEnabled(enabled bool) {
	if enabled {
		SetRawMode()
	} else {
		Restore()
	}
}

func SetRawMode() {
	var a syscall.Termios
	if err := termios.Tcgetattr(uintptr(fd), &a); err != nil {
		fmt.Println("Failed to get attribute")
		return
	}
	termios.Cfmakeraw(&a)
	termios.Tcsetattr(uintptr(fd), termios.TCSANOW, (*syscall.Termios)(&a))
}

func Restore() {
	termios.Tcsetattr(uintptr(fd), termios.TCIOFLUSH, &orig)
}
