package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

var _TIOCGWINSZ int

type WinSize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func init() {
	if runtime.GOOS == "darwin" {
		_TIOCGWINSZ = 1074295912
	} else {
		_TIOCGWINSZ = 0x5413
	}
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
	fmt.Print("\033[K")
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

func RestoreShell() {
	SetRawModeEnabled(false)
	SetCursorVisible(true)
	ToggleAlternateScreen(false)
}

func PrintCommands(winSize *WinSize) {
	PositionCursor(0, int(winSize.Row)-2)
	optionWidth := int((winSize.Col - 4) / 4)
	nextPage := Truncate(WhiteBackground("N")+" Next Page", optionWidth, true)
	prevPage := Truncate(WhiteBackground("P")+" Previous Page", optionWidth, true)
	back := Truncate(WhiteBackground("B")+" Back", optionWidth, true)
	goTo := Truncate(WhiteBackground("G")+" Go To", optionWidth, false)
	openInBrowser := Truncate(WhiteBackground("O")+" Open in Browser", optionWidth, true)
	copyURL := Truncate(WhiteBackground("U")+" Copy URL", optionWidth, true)
	copyText := Truncate(WhiteBackground("T")+" Copy Text", optionWidth, true)
	quit := Truncate(WhiteBackground("Q")+" Quit", optionWidth, false)

	fmt.Printf("  %s%s%s%s\n  %s%s%s%s\n", nextPage, prevPage, back, goTo, openInBrowser, copyURL, copyText, quit)
}

func PrintCentered(str string, winSize *WinSize) {
	lines := strings.Split(str, "\n")
	var lineLen int
	if len(lines) == 0 {
		lineLen = len(str)
		lines = append(lines, str)
	} else {
		lineLen = len(lines[0])
	}

	nEmpty := (int(winSize.Row) - (5 + len(lines))) / 2
	fmt.Print(EmptyLines(nEmpty))
	for _, s := range lines {
		fmt.Print(Spaces((int(winSize.Col)-lineLen)/2) + s + "\n")
	}
}

func PrintGoTo() {
	fmt.Print("Go to: ")
}

func MarkLinks(paragraphText string, options []Ref) string {
	var buffer bytes.Buffer
	for _, option := range options {
		index := strings.Index(paragraphText, option.text)
		if index >= 0 {
			buffer.WriteString(paragraphText[:index] + Underline(option.text))
			paragraphText = paragraphText[index+len(option.text):]
		}
	}
	buffer.WriteString(paragraphText)
	paragraphText = buffer.String()
	return paragraphText
}

func WikishellAscii() (str string) {
	str = "           _ _    _     _          _ _\n" +
		"          (_) |  (_)   | |        | | |\n" +
		" __      ___| | ___ ___| |__   ___| | |\n" +
		" \\ \\ /\\ / / | |/ / / __| '_ \\ / _ \\ | |\n" +
		"  \\ V  V /| |   <| \\__ \\ | | |  __/ | |\n" +
		"   \\_/\\_/ |_|_|\\_\\_|___/_| |_|\\___|_|_|\n"
	return

}

func GetHrefValue(s *goquery.Selection) (string, bool) {
	val, exists := s.Attr("href")
	if !exists {
		return val, false
	}

	titleAttr, titleExists := s.Attr("title")
	if (strings.HasPrefix(val, "/wiki/") || strings.HasPrefix(val, "/w/")) && len(s.Text()) > 1 &&
		(!titleExists || (!strings.HasPrefix(titleAttr, "Help:IPA") && !strings.HasPrefix(titleAttr, "Wikipedia:") && !strings.HasPrefix(titleAttr, "File:") && !strings.Contains(val, "redlink=1"))) {
		return val, true
	} else {
		return val, false
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

func TruncateParagraph(paragraphText string, options []Ref, winSize *WinSize) (string, int) {
	maxOptions := Min(10, len(options))
	length := len(paragraphText)
	escapeLength := Min(10, len(options)) * 14
	rows := (length - escapeLength) / int(winSize.Col-5)
	availableRows := int(winSize.Row) - 9

	if rows > availableRows {
		maxOptions = 0
		toIndex := Max(0, Min(length-1, length-(rows-availableRows+1)*int(winSize.Col-5)))
		paragraphText = paragraphText[:toIndex] + "\033[0m..."
	} else {
		maxOptions = Min(maxOptions, availableRows-rows)
	}
	return paragraphText, maxOptions
}

func IsParagraphValid(text string) bool {
	if len(text) == 0 {
		return false
	}
	if strings.HasPrefix(text, "Coordinates: ") {
		return false
	}
	return true
}

func Spaces(n int) string {
	return RepeatString(n, " ")
}

func EmptyLines(n int) string {
	return RepeatString(n, "\n")
}

func RepeatString(n int, str string) string {
	if n <= 0 {
		return ""
	}
	var buffer bytes.Buffer
	for i := 0; i < n; i++ {
		buffer.WriteString(str)
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

func RemoveBrackets(s string) string {
	reg, err := regexp.Compile("\\[([0-9]+|citation needed)\\]")
	if err != nil {
		log.Fatal(err)
	}
	modified := reg.ReplaceAllString(s, "")
	return modified
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
