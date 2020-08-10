// lnd, main program
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/RaymondJiangkw/Lazy/lazyNovelDownloader/search"

	"github.com/RaymondJiangkw/Lazy/lazyNovelDownloader/extract"
	"github.com/RaymondJiangkw/Lazy/lazyNovelDownloader/write"
)

var novelName = flag.String("name", "", "[compulsory] Novel Name")
var novelAuthor = flag.String("author", "", "[optional] Novel Author")
var outputFileName = flag.String("o", "", `[optional] Output File Name(can include path)`)
var outputFileFormat = flag.String("format", "txt", "[optional] txt/epub")
var catalogURL = flag.String("source", "", "[optional] URL for Catalog Html File of Novel")
var autoDetection = flag.Bool("auto", false, "[optional] Whether to detect catalogs automatically, given the name of novel")

const (
	invalidPrompt = "Invalid Arguments! One of [source] and [auto] must be specified. Type in -help/-h for help."
	errorPrompt   = ", encounter Error %v. The Program is terminated unexpectedly."
)

func main() {
	flag.Parse()
	if len(flag.Args()) > 0 || (*outputFileFormat != "txt" && *outputFileFormat != "epub") || *novelName == "" || (*catalogURL == "" && !*autoDetection) {
		log.Fatalf("%s", invalidPrompt)
	}
	if *outputFileName == "" {
		*outputFileName = *novelName
	}
	var c_s []extract.Chapters
	var errs []error
	if *catalogURL != "" {
		c_s, errs = extract.Extract(os.Stdout, []string{*catalogURL}, *novelName, false, false)
		if errs[0] != nil {
			log.Fatalf("While extracting contents"+errorPrompt, errs[0])
		}
	} else { // Auto-Detection
		urls, err := search.Search(os.Stdout, *novelName)
		if err != nil {
			log.Fatalf("While searching for catalogs"+errorPrompt, err)
		}
		c_s, errs = extract.Extract(os.Stdout, urls, *novelName, true, true)
		if errs[0] != nil {
			log.Fatalf("While extracting contents"+errorPrompt, errs[0])
		}
	}

	var err error
	*outputFileName, err = filepath.Abs(*outputFileName)
	if err != nil {
		log.Fatalf("While getting output file path"+errorPrompt, err)
	}
	switch *outputFileFormat {
	case "txt":
		err = write.WriteToTxt(os.Stdout, c_s[0], *outputFileName, write.NovelInfo{Name: *novelName, Author: *novelAuthor})
	case "epub":
		err = write.WriteToEpub(os.Stdout, c_s[0], *outputFileName, write.NovelInfo{Name: *novelName, Author: *novelAuthor})
	}
	if err != nil {
		log.Fatalf("While writing to file"+errorPrompt, err)
	}
}
