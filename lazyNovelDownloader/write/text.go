// text convert Chapters to .txt file
package write

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/RaymondJiangkw/Lazy/utils"

	"github.com/bmaupin/go-epub"

	"github.com/RaymondJiangkw/Lazy/lazyNovelDownloader/extract"
)

const (
	Prologue = `	These texts are generated by lazyNovelDownloader, which is developed by RaymondJiangkw.
	This software does not provide any text by itself. Rights of texts are all reserved for original authors, who have the rights to require users to delete their works.
	Any conflict, dispute and lawsuit resulted from texts are all attributed to users. RaymondJiangkw does not take any responsibility of it.
  
	Copyright © 2020 RaymondJiangkw. All rights reserved.`
	Lack     = `Lack Available Content!`
	cssFile  = "epub.css"
	fontFile = "Kaiti.ttf"
)

const (
	outputPrePostfixText = "    "
	outputIOText         = "I/O Processing..."
)

type NovelInfo struct {
	Name   string
	Author string
}

func WriteToTxt(writer io.Writer, chapters extract.Chapters, filePath string, novelInfo NovelInfo) (e error) {
	var display utils.Display
	signal := make(chan struct{})
	if !strings.HasSuffix(filePath, ".txt") {
		filePath += ".txt"
	}
	fmt.Printf("Writing to file %s...\n", filepath.Base(filePath))
	novel := Prologue + "\n"
	novel += "Name:	" + novelInfo.Name + "\n"
	novel += "Author:	" + novelInfo.Author + "\n"

	go display.ProgressBar(&utils.ProgressBarOption{Writer: writer, Phase: []int{1}, Signal: [][]<-chan struct{}{[]<-chan struct{}{signal}}, Maximum: [][]int{[]int{len(chapters)}}, Prefix: [][]string{[]string{outputPrePostfixText + "Write: "}}, Postfix: [][]string{[]string{outputPrePostfixText}}})

	for _, c := range chapters {
		novel += "\n" + c.Name + "\n"
		if !c.Fetch {
			novel += Lack
		} else {
			novel += c.Content
		}
		novel += "\n"
		signal <- struct{}{}
	}
	fmt.Fprintf(writer, "%s", outputIOText)
	e = utils.WriteFileString(filePath, &novel, false)
	return
}

func WriteToEpub(writer io.Writer, chapters extract.Chapters, filePath string, novelInfo NovelInfo) (e error) {
	var display utils.Display
	signal := make(chan struct{})
	if !strings.HasPrefix(filePath, ".epub") {
		filePath += ".epub"
	}
	fmt.Printf("Writing to file %s...\n", filepath.Base(filePath))
	novelName := novelInfo.Name
	epub := epub.NewEpub(novelName)
	// Set Novel Information
	epub.SetAuthor(novelInfo.Author)
	epub.SetDescription(Prologue)
	// Set CSS and Font
	cssPath, _ := filepath.Abs(cssFile)
	fontPath, _ := filepath.Abs(fontFile)
	epub.AddFont(fontPath, fontFile)
	epub.AddCSS(cssPath, cssFile)
	epub.AddSection(EpubFormatString(Prologue), "Prologue", "prologue.xhtml", "")
	// Generate Catalog
	catalog := `<h1>` + `Catalog` + `</h1>`
	catalog += `<div id="catalog">`
	for finish, c := range chapters {
		catalog += `<a href="` + strconv.Itoa(finish) + `.xhtml">` + c.Name + `</a><br />`
	}
	catalog += `</div>`
	epub.AddSection(catalog, "Catalog", "catalog.xhtml", "")

	go display.ProgressBar(&utils.ProgressBarOption{Writer: writer, Phase: []int{1}, Signal: [][]<-chan struct{}{[]<-chan struct{}{signal}}, Maximum: [][]int{[]int{len(chapters)}}, Prefix: [][]string{[]string{outputPrePostfixText + "Write: "}}, Postfix: [][]string{[]string{outputPrePostfixText}}})

	for finish, c := range chapters {
		content := c.Content
		if !c.Fetch {
			content = Lack
		}
		_, e = epub.AddSection(`<h2>`+c.Name+`</h2>`+`<div id="content">`+EpubFormatString(content)+`</div>`+`<div id="foot">`+`<a href="catalog.xhtml" align="right">Back to Catalog</a>`+`</div>`, c.Name, strconv.Itoa(finish)+".xhtml", "")
		if e != nil {
			close(signal)
			return
		}
		signal <- struct{}{}
	}
	close(signal)
	fmt.Fprintf(writer, "%s", outputIOText)
	e = epub.Write(filePath)
	return
}

func EpubFormatString(s string) (ret string) {
	r := bufio.NewScanner(strings.NewReader(s))
	for r.Scan() {
		ret += `<p>` + r.Text() + `</p>`
	}
	return
}
