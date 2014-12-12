package main

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/atotto/clipboard"
	"github.com/skratchdot/open-golang/open"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

const (
	maxWidth   = 256
	leftMargin = "  "
)

type Ref struct {
	url  string
	text string
}

var docStack *Stack
var currentArticle *Article
var winResizeChan chan os.Signal
var stdinChan chan []byte
var errorChan chan error

// var logfile *os.File

func main() {

	SetRawModeEnabled(true)
	ToggleAlternateScreen(true)
	SetCursorVisible(false)

	defer func() {
		RestoreShell()
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
		QueryArticle(query)
	}
}

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

func QueryArticle(query string) bool {
	doc, err := goquery.NewDocument("http://en.wikipedia.org/wiki/" + query)
	if err != nil {
		log.Fatal(err)
	}

	validSel := doc.Find(".mw-content-ltr")
	if validSel.Length() == 0 {
		return HandleNotFound()
	} else {
		disambigBoxLength := doc.Find(".dmbox-disambig").Length()
		article := NewArticle(doc, 0, disambigBoxLength > 0)
		HandleArticle(article)
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
	HandleUserInput(nil, nil, true)
}

func HandleArticle(article *Article) {

	ClearScrean()
	SetCursorVisible(false)

	winSize, _ := GetWinsize()
	validSel := article.doc.Find(".mw-content-ltr")

	var contentSel *goquery.Selection
	var paragraphs *goquery.Selection

	if !article.isDisambiguous {
		paragraphs = validSel.Children().Filter("p").NotFunction(func(i int, sel *goquery.Selection) bool {
			return (len(sel.Text()) < 2)
		})
		contentSel = paragraphs.Eq(article.paragraphIndex)
		if !IsParagraphValid(contentSel.Text()) && paragraphs.Length() > article.paragraphIndex+1 {
			article.paragraphIndex += 1
			HandleArticle(article)
			return
		}
		article.numberOfPages = paragraphs.Length()
	} else {
		numberOfListItems := article.GetLinkListItems().Length()
		article.numberOfPages = int(math.Ceil(float64(numberOfListItems) / 10))
	}

	options := article.Process(contentSel, winSize)

	currentArticle = article
	HandleUserInput(contentSel, options, false)
}

func HandleUserInput(contentSel *goquery.Selection, options []Ref, isInputMode bool) {

	if !isInputMode {
		if currentArticle == nil {
			return
		}
	}

	var query string
	isReadingQuery := false
	shouldOverwrite := false

	for {
		b, resized := ReadChar()
		if resized {
			if isInputMode {
				HandleEmptyQuery("")
			} else if currentArticle != nil {
				HandleArticle(currentArticle)
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
				if currentArticle != nil {
					HandleArticle(currentArticle)
					return
				} else {
					OverwriteCurrentLine(Spaces(24), true)
					SetCursorVisible(false)
					isInputMode = false
					break
				}
			case s == "\n":
				words := strings.Split(query, " ")
				if !isInputMode && currentArticle != nil {
					docStack.Push(currentArticle)
				}
				if !QueryArticle(BuildQuery(words)) {
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
				open.Run(currentArticle.doc.Url.String())
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
				if NextPage(currentArticle) {
					return
				}
			case s == "p":
				if PreviousPage(currentArticle) {
					return
				}
			case s == "b":
				if PopArticle() {
					return
				} else {
					Alert()
				}
			case s == "u":
				clipboard.WriteAll(currentArticle.doc.Url.String())
				break
			case s == "t":
				if currentArticle.isDisambiguous || contentSel == nil {
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
						docStack.Push(currentArticle)
						selectedArticle := NewArticle(selectedDoc, 0, false)
						HandleArticle(selectedArticle)
						return
					} else {
						Alert()
					}
				}
			}
		}
	}
}

func NextPage(article *Article) bool {
	if article.numberOfPages > article.paragraphIndex+1 {
		docStack.Push(article)
		next := NewArticle(article.doc, article.paragraphIndex+1, article.isDisambiguous)
		HandleArticle(next)
		return true
	} else {
		Alert()
		return false
	}
}

func PreviousPage(article *Article) bool {
	if article.paragraphIndex > 0 {
		docStack.Push(article)
		prev := NewArticle(article.doc, article.paragraphIndex-1, article.isDisambiguous)
		HandleArticle(prev)
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

func PopArticle() bool {
	prevItem, exists := docStack.Pop()
	if exists {
		article := prevItem.(*Article)
		HandleArticle(article)
		return true
	} else {
		return false
	}
}
