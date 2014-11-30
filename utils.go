package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

const (
	_TIOCGWINSZ = 1074295912 // on linux use 0x5413, On OSX use 1074295912
)

type WinSize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func GetWinsize() (*WinSize, error) {
	wsize := new(WinSize)

	r1, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(_TIOCGWINSZ),
		uintptr(unsafe.Pointer(wsize)),
	)

	if int(r1) == -1 {
		wsize.Col = 80
		wsize.Row = 25
		return wsize, os.NewSyscallError("GetWinsize", errors.New("Failed to get Winsize "+string(int(errno))))
	}
	return wsize, nil
}

func ClearScrean() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	c.Run()
}

func OverwriteCurrentLine(str string, positionCursorAtBeginning bool) {
	winSize, _ := GetWinsize()
	PositionCursor(0, int(winSize.Row))
	fmt.Print(str)
	if positionCursorAtBeginning {
		PositionCursor(0, int(winSize.Row))
	}
}

func PositionCursor(x, y int) {
	fmt.Printf("\033[%d;%dH", y, x)
}

func ToggleAlternateScreen(onoff bool) {
	if onoff {
		fmt.Print("\033[?1049h")
	} else {
		fmt.Print("\033[?1049l")
	}
}

func SetCursorVisible(visible bool) {
	if visible {
		fmt.Print("\033[?25h")
	} else {
		fmt.Print("\033[?25l")
	}
}

func Truncate(str string, width int, pad bool) string {
	strLen := len(str)
	strLenStripped := strLen - 15
	if strLenStripped > width-1 {
		truncation := Max(16, strLen-3-(strLenStripped-width))
		newStr := str[:int(truncation)] + "..."
		return newStr
	} else if pad {
		return str + Spaces(width-strLenStripped)
	} else {
		return str
	}
}

func Spaces(n int) string {
	if n <= 0 {
		return ""
	}
	var buffer bytes.Buffer
	for i := 0; i < n; i++ {
		buffer.WriteString(" ")
	}
	return buffer.String()
}

func WhiteBackground(str string) string {
	return "\033[107m\033[30m" + str + "\033[0m"
}

func Bold(str string) string {
	return "\033[1m" + str + "\033[0m"
}

func Underline(str string) string {
	return "\033[38;5;27m" + str + "\033[0m"
}

func Backspace() {
	fmt.Print("\033[1D \033[1D")
}

func Alert() {
	fmt.Print("\x07")
}

func Max(x, y int) int {
	if x > y {
		return x
	} else {
		return y
	}
}

func Min(x, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}