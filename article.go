package main

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"strings"
)

type Article struct {
	doc            *goquery.Document
	paragraphIndex int
	isDisambiguous bool
	numberOfPages  int
	title          string
}

func NewArticle(doc *goquery.Document, paragraphIndex int, isDisambiguous bool) (article *Article) {
	article = new(Article)
	article.doc = doc
	article.paragraphIndex = paragraphIndex
	article.isDisambiguous = isDisambiguous
	return article
}

func (article *Article) Process(content *goquery.Selection, winSize *WinSize) []Ref {
	if article.numberOfPages > 0 {
		article.PrintTitle()
	}

	var options []Ref
	if article.isDisambiguous {
		options = article.PrintDisambiguationLinks(winSize)
	} else {
		options = article.PrintParagraph(content, winSize)
	}

	fmt.Println()

	PrintCommands(winSize)

	return options
}

func (article *Article) PrintTitle() {
	titleSel := article.doc.Find(".firstHeading")
	title := titleSel.Text()
	fmt.Printf("\n  %s\t[%d/%d]\n\n", Bold(strings.ToUpper(title)), article.paragraphIndex+1, article.numberOfPages)
	article.title = title
}

func (article *Article) PrintParagraph(contentSel *goquery.Selection, winSize *WinSize) []Ref {

	paragraphText := contentSel.Text()
	options := []Ref{}

	if article.paragraphIndex == 0 {
		disambig, exists := article.FindOtherUsesRef()
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

func (article *Article) PrintDisambiguationLinks(winSize *WinSize) (options []Ref) {

	fmt.Printf("  Articles associated with the title '%s':\n\n", article.title)

	fromPage := article.paragraphIndex * 10
	toPage := fromPage + 10
	maxColumnWidth := int(winSize.Col) - 8
	if maxColumnWidth > maxWidth {
		maxColumnWidth = maxWidth
	}
	article.GetLinkListItems().Each(func(i int, s *goquery.Selection) {
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

func (article *Article) GetLinkListItems() *goquery.Selection {
	return article.doc.Find(".mw-content-ltr").Find("li").NotFunction(func(i int, sel *goquery.Selection) bool {
		val, exists := sel.Attr("class")
		if exists && strings.HasPrefix(val, "toclevel") {
			return true
		} else {
			return false
		}
	})
}

func (article *Article) FindOtherUsesRef() (Ref, bool) {
	disambig := article.doc.Find(".hatnote").Find(".mw-disambig")
	if disambig.Length() > 0 {
		val, exists := disambig.Attr("href")
		if exists {
			return Ref{val, "Other uses of " + Bold(article.title) + " (disambiguation)"}, true
		}
	}
	return Ref{}, false
}
