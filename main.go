package main

/*
#include "termutil.h"
*/
import "C"

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/atotto/clipboard"
	"github.com/skratchdot/open-golang/open"
	"log"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const (
	maxWidth   = 256
	leftMargin = "  "
)

type StackItem struct {
	doc            *goquery.Document
	paragraphIndex int
	isDisamiguous  bool
}

type Ref struct {
	url  string
	text string
}

var docStack *Stack
var currentArticle *StackItem
var winResizeChan chan os.Signal
var stdinChan chan []byte
var errorChan chan error

// var logfile *os.File

func main() {

	C.rawmodeon()
	ToggleAlternateScreen(true)
	SetCursorVisible(false)

	// logfile, _ = os.Create("log")

	defer func() {
		RestoreShell()
		// logfile.Sync()
		// logfile.Close()
	}()

	stdinChan = make(chan []byte)
	errorChan = make(chan error)
	winResizeChan = make(chan os.Signal)

	StartListening()
	ListenToOSSignals()

	docStack = new(Stack)

	if len(os.Args) <= 1 {
		HandleEmptyQuery(WikishellAscii())
	} else {
		args := os.Args[1:]
		query := BuildQuery(args)
		QueryAbstract(query)
	}
}

// func logToFile(str string) {
// 	logfile.WriteString(str + "\n")
// 	logfile.Sync()
// }

func BuildQuery(parts []string) string {
	var buffer bytes.Buffer
	for i, s := range parts {
		if i > 0 {
			buffer.WriteString("_")
		}
		buffer.WriteString(strings.Title(s))
	}
	return buffer.String()
}

func QueryAbstract(query string) bool {
	doc, err := goquery.NewDocument("http://en.wikipedia.org/wiki/" + query)
	if err != nil {
		log.Fatal(err)
	}

	validSel := doc.Find(".mw-content-ltr")
	if validSel.Length() == 0 {
		return HandleNotFound()
	} else {
		disambigBoxLength := doc.Find(".dmbox-disambig").Length()
		HandleArticle(doc, 0, disambigBoxLength > 0)
		return true
	}
}

func HandleNotFound() bool {
	if currentArticle != nil {
		return false
	} else {
		HandleEmptyQuery(WikishellAscii())
		return true
	}
}

func HandleEmptyQuery(title string) {
	ClearScrean()
	winSize, _ := GetWinsize()
	PrintCentered(title, winSize)
	PrintCommands(winSize)
	PrintGoTo()
	SetCursorVisible(true)
	HandleUserInput(0, nil, nil, true)
}

func HandleArticle(doc *goquery.Document, paragraphIndex int, isDisamiguous bool) {

	ClearScrean()
	SetCursorVisible(false)

	winSize, _ := GetWinsize()
	validSel := doc.Find(".mw-content-ltr")

	var numberOfPages int
	var contentSel *goquery.Selection
	var paragraphs *goquery.Selection

	if !isDisamiguous {
		paragraphs = validSel.Children().Filter("p").NotFunction(func(i int, sel *goquery.Selection) bool {
			return (len(sel.Text()) < 2)
		})
		contentSel = paragraphs.Eq(paragraphIndex)
		if !IsValid(contentSel.Text()) && paragraphs.Length() > paragraphIndex+1 {
			HandleArticle(doc, paragraphIndex+1, isDisamiguous)
			return
		}
		numberOfPages = paragraphs.Length()
	} else {
		numberOfListItems := GetLinkListItems(validSel).Length()
		numberOfPages = int(math.Ceil(float64(numberOfListItems) / 10))
	}

	var docTitle string
	if numberOfPages > 0 {
		docTitle = PrintTitle(doc, paragraphIndex+1, numberOfPages)
	} else {
		docTitle = ""
	}

	var options []Ref
	if isDisamiguous {
		options = PrintDisambiguationLinks(validSel, paragraphIndex, docTitle)
	} else {
		options = PrintParagraph(doc, paragraphIndex, contentSel, docTitle, winSize)
	}

	fmt.Println()

	PrintCommands(winSize)
	SetCurrentArticle(doc, paragraphIndex, isDisamiguous)
	HandleUserInput(numberOfPages, contentSel, options, false)
}

func PrintTitle(doc *goquery.Document, paragraphIndex int, numOfParagraphs int) string {
	titleSel := doc.Find(".firstHeading")
	title := titleSel.Text()
	fmt.Printf("\n  %s\t[%d/%d]\n\n", Bold(strings.ToUpper(title)), paragraphIndex, numOfParagraphs)
	return title
}

func PrintParagraph(doc *goquery.Document, paragraphIndex int, contentSel *goquery.Selection, title string, winSize *WinSize) []Ref {

	paragraphText := contentSel.Text()
	options := []Ref{}

	if paragraphIndex == 0 {
		disambig, exists := FindOtherUsesRef(doc, title)
		if exists {
			options = append(options, disambig)
		}
	}

	contentSel.Find("a").Each(func(i int, s *goquery.Selection) {
		if val, valid := GetHrefValue(s); valid {
			options = append(options, Ref{val, s.Text()})
		}
	})

	var buffer bytes.Buffer
	words := strings.Split(paragraphText, " ")
	buffer.WriteString(leftMargin)
	currentLength := 0
	maxColumnWidth := Min(int(winSize.Col)-4, maxWidth)

	for _, word := range words {
		wordLength := len(word)
		if (currentLength + wordLength) > maxColumnWidth {
			buffer.WriteString("\n  ")
			currentLength = 0
		}
		buffer.WriteString(word)
		buffer.WriteString(" ")
		currentLength += wordLength + 1
	}

	paragraphText = buffer.String()
	paragraphText = RemoveBrackets(paragraphText)
	buffer.Reset()

	paragraphText = MarkLinks(paragraphText, options)
	paragraphToPrint, maxOptions := TruncateParagraph(paragraphText, options, winSize)

	fmt.Println(paragraphToPrint)
	fmt.Println()

	for i, ref := range options[:maxOptions] {
		fmt.Printf("  (%d)\t%s\n", (i+1)%10, ref.text)
	}
	return options
}

func PrintDisambiguationLinks(doc *goquery.Selection, pageIndex int, docTitle string) (options []Ref) {

	fmt.Printf("  Articles associated with the title '%s':\n\n", docTitle)

	fromPage := pageIndex * 10
	toPage := fromPage + 10
	winSize, _ := GetWinsize()
	maxColumnWidth := int(winSize.Col) - 8
	if maxColumnWidth > maxWidth {
		maxColumnWidth = maxWidth
	}
	GetLinkListItems(doc).Each(func(i int, s *goquery.Selection) {
		if i < fromPage || i >= toPage {
			return
		}
		ahref := s.Find("a")
		if ahref != nil {
			val, exists := ahref.Attr("href")
			if exists && (strings.HasPrefix(val, "/wiki/") || strings.HasPrefix(val, "/w/")) {
				children := s.Children()
				var displayText string
				if children.Length() > 1 {
					displayText = s.Children().First().Text()
				} else {
					displayText = s.Text()
				}
				if len(displayText) > maxColumnWidth {
					displayText = displayText[:(maxColumnWidth-3)] + "..."
				}
				fmt.Printf("  (%d)\t%s\n", (i+1)%10, displayText)
				options = append(options, Ref{val, strings.Title(s.Text())})
			}
		}
	})
	return
}

func FindOtherUsesRef(doc *goquery.Document, title string) (Ref, bool) {
	disambig := doc.Find(".hatnote").Find(".mw-disambig")
	if disambig.Length() > 0 {
		val, exists := disambig.Attr("href")
		if exists {
			return Ref{val, "Other uses of " + Bold(title) + " (disambiguation)"}, true
		}
	}
	return Ref{}, false
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

func PrintGoTo() {
	fmt.Print("Go to: ")
}
func HandleUserInput(numberOfPages int, contentSel *goquery.Selection, options []Ref, isInputMode bool) {

	var doc *goquery.Document
	var paragraphIndex int
	var isDisamiguous bool

	if !isInputMode {
		if currentArticle == nil {
			return
		}
		doc = currentArticle.doc
		paragraphIndex = currentArticle.paragraphIndex
		isDisamiguous = currentArticle.isDisamiguous
	}

	isReadingQuery := false
	var query string
	shouldOverwrite := false

	for {
		b, resized := ReadChar()
		if resized {
			if isInputMode {
				HandleEmptyQuery("")
			} else {
				HandleArticle(doc, paragraphIndex, isDisamiguous)
			}
			break
		}

		s := string(b)

		if isReadingQuery || isInputMode {

			// Char input mode

			if shouldOverwrite {
				shouldOverwrite = false
				OverwriteCurrentLine(Spaces(24), true)
				PrintGoTo()
			}

			switch {
			case b == 27:
				isReadingQuery = false
				query = ""
				if doc != nil {
					HandleArticle(doc, paragraphIndex, isDisamiguous)
					return
				} else {
					OverwriteCurrentLine(Spaces(24), true)
					SetCursorVisible(false)
					isInputMode = false
					break
				}
			case s == "\n":
				words := strings.Split(query, " ")
				if !isInputMode && doc != nil {
					docStack.Push(&StackItem{doc, paragraphIndex, isDisamiguous})
				}
				if !QueryAbstract(BuildQuery(words)) {
					query = ""
					OverwriteCurrentLine("Article not found.", false)
					Alert()
					shouldOverwrite = true
				} else {
					return
				}
			case b == 127 || b == 8:
				if len(query) > 0 {
					Backspace()
					query = query[:len(query)-1]
				} else {
					Alert()
				}
			default:
				fmt.Print(s)
				query += s
			}
		} else {

			// Readline input mode

			s = strings.ToLower(s)
			switch {
			case s == "o":
				open.Run(doc.Url.String())
				break
			case s == "q":
				return
			case s == "g":
				PrintGoTo()
				SetCursorVisible(true)
				query = ""
				isReadingQuery = true
				break
			case s == "n" || s == "\n":
				if NextPage(doc, numberOfPages, paragraphIndex, isDisamiguous) {
					return
				}
			case s == "p":
				if PreviousPage(doc, paragraphIndex, isDisamiguous) {
					return
				}
			case s == "b":
				if PopArticle() {
					return
				} else {
					Alert()
				}
			case s == "u":
				clipboard.WriteAll(doc.Url.String())
				break
			case s == "t":
				if isDisamiguous || contentSel == nil {
					Alert()
				} else {
					clipboard.WriteAll(contentSel.Text())
				}
				break
			default:
				integer, err := strconv.Atoi(s)
				if err != nil {
					Alert()
				} else {
					if selectedDoc, succeeded := HandleRefSelection(integer, options); succeeded {
						docStack.Push(&StackItem{doc, paragraphIndex, isDisamiguous})
						HandleArticle(selectedDoc, 0, false)
						return
					} else {
						Alert()
					}
				}
			}
		}
	}
}

func NextPage(doc *goquery.Document, numberOfPages int, paragraphIndex int, isDisamiguous bool) bool {
	if numberOfPages > paragraphIndex+1 {
		docStack.Push(&StackItem{doc, paragraphIndex, isDisamiguous})
		HandleArticle(doc, paragraphIndex+1, isDisamiguous)
		return true
	} else {
		Alert()
		return false
	}
}

func PreviousPage(doc *goquery.Document, paragraphIndex int, isDisamiguous bool) bool {
	if paragraphIndex > 0 {
		docStack.Push(&StackItem{doc, paragraphIndex, isDisamiguous})
		HandleArticle(doc, paragraphIndex-1, isDisamiguous)
		return true
	} else {
		Alert()
		return false
	}
}

func HandleRefSelection(index int, options []Ref) (*goquery.Document, bool) {
	i := index
	if i == 0 {
		i += 10
	}
	if i >= 0 && i <= len(options) {
		selected := options[i-1].url
		url := "http://en.wikipedia.org" + selected
		articleDoc, err := goquery.NewDocument(url)
		if err != nil {
			log.Fatal(err)
			return nil, false
		}
		return articleDoc, true
	} else {
		return nil, false
	}
}

func GetLinkListItems(sel *goquery.Selection) *goquery.Selection {
	return sel.Find("li").NotFunction(func(i int, sel *goquery.Selection) bool {
		val, exists := sel.Attr("class")
		if exists && strings.HasPrefix(val, "toclevel") {
			return true
		} else {
			return false
		}
	})
}

func PopArticle() bool {
	prevItem, exists := docStack.Pop()
	if exists {
		stackItem := prevItem.(*StackItem)
		HandleArticle(stackItem.doc, stackItem.paragraphIndex, stackItem.isDisamiguous)
		return true
	} else {
		return false
	}
}

func StartListening() {
	signal.Notify(winResizeChan, syscall.SIGWINCH)

	go func(ch chan []byte, eCh chan error) {
		for {
			b := make([]byte, 1)
			_, err := os.Stdin.Read(b)
			if err != nil {
				eCh <- err
				return
			}
			ch <- b
		}
	}(stdinChan, errorChan)
}

func ReadChar() (byte, bool) {
	select {
	case <-winResizeChan:
		return 0, true
		break
	case b := <-stdinChan:
		return b[0], false
		break
	case err := <-errorChan:
		fmt.Println(err)
		return 0, false
		break
	}
	return 0, false
}

func ListenToOSSignals() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			fmt.Println(sig)
			RestoreShell()
			os.Exit(0)
		}
	}()
}

func ListenToWinChange() {
	c := make(chan os.Signal, 20)
	signal.Notify(c, syscall.SIGWINCH)
	go func() {
		for sig := range c {
			fmt.Println(sig)
		}
	}()
}

func RestoreShell() {
	C.rawmodeoff()
	SetCursorVisible(true)
	ToggleAlternateScreen(false)
}

func IsValid(text string) bool {
	if len(text) == 0 {
		return false
	}
	if strings.HasPrefix(text, "Coordinates: ") {
		return false
	}
	return true
}

func GetHrefValue(s *goquery.Selection) (string, bool) {
	val, exists := s.Attr("href")
	titleAttr, titleExists := s.Attr("title")
	if exists && (strings.HasPrefix(val, "/wiki/") || strings.HasPrefix(val, "/w/")) && len(s.Text()) > 1 &&
		(!titleExists || (!strings.HasPrefix(titleAttr, "Help:IPA") && !strings.HasPrefix(titleAttr, "Wikipedia:Citation needed"))) {
		return val, true
	} else {
		return val, false
	}
}

func SetCurrentArticle(doc *goquery.Document, paragraphIndex int, isDisamiguous bool) {
	if currentArticle == nil {
		currentArticle = new(StackItem)
	}
	currentArticle.doc = doc
	currentArticle.paragraphIndex = paragraphIndex
	currentArticle.isDisamiguous = isDisamiguous
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
