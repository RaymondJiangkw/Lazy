// lnd, main program
package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/RaymondJiangkw/Lazy/lazyNovelDownloader/write"

	"github.com/RaymondJiangkw/Lazy/lazyNovelDownloader/extract"
)

var novelName = flag.String("name", "", "[compulsory] Novel Name")
var novelAuthor = flag.String("author", "", "[optional] Novel Author")
var outputFileName = flag.String("o", "", `[optional] Output File Name(can include path)`)
var outputFileFormat = flag.String("format", "txt", "[optional] txt/epub")
var catalogURL = flag.String("source", "", "[compulsory] URL for Catalog Html File of Novel")

const (
	invalidPrompt = "Invalid Arguments! Type in -help/-h for help."
	errorPrompt   = ", encounter Error %v. The Program is terminated unexpectedly."
)

func main() {
	flag.Parse()
	if len(flag.Args()) > 0 || (*outputFileFormat != "txt" && *outputFileFormat != "epub") || *novelName == "" || *catalogURL == "" {
		fmt.Println(invalidPrompt)
	}
	if *outputFileName == "" {
		*outputFileName = *novelName
	}
	c_s, err := extract.Extract(*catalogURL, *novelName)
	if err != nil {
		log.Fatalf("While extracting contents"+errorPrompt, err)
	}
	*outputFileName, err = filepath.Abs(*outputFileName)
	if err != nil {
		log.Fatalf("While getting output file path"+errorPrompt, err)
	}
	switch *outputFileFormat {
	case "txt":
		err = write.WriteToTxt(c_s, *outputFileName, write.NovelInfo{Name: *novelName, Author: *novelAuthor})
	case "epub":
		err = write.WriteToEpub(c_s, *outputFileName, write.NovelInfo{Name: *novelName, Author: *novelAuthor})
	}
	if err != nil {
		log.Fatalf("While writing to file"+errorPrompt, err)
	}
}
