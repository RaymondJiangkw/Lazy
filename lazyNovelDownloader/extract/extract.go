// extract extract fiction from website in the format of []Chapter
package extract

import (
	"fmt"
	"io"
	"time"

	"github.com/RaymondJiangkw/Lazy/utils"
)

const (
	outputPrePostfixEachTurn = "    "
	extractCacheFolder       = ".novel"
)

func Extract(writer io.Writer, urls []string, novelName string) ([]Chapters, []error) {
	var display utils.Display
	cnt := len(urls)
	catalogueSignal := make(chan struct{}, cnt)
	catalogues := make([]Chapters, cnt)
	catalogueErrors := make([]error, cnt)

	for i, url := range urls {
		go func(i int, url string) {
			defer func() {
				catalogueSignal <- struct{}{}
			}()
			catalogues[i], catalogueErrors[i] = Catalogue(url)
		}(i, url)
	}
	display.EasyProgress(writer, "Fetching Catalogues", "...", len(urls), catalogueSignal)
	close(catalogueSignal)
	beginTime := time.Now()
	var times int
	for {
		times++
		fmt.Fprintf(writer, "%dth Turn:\n", times)
		initSignal := make(chan struct{})
		go display.TemporaryText(writer, "Initializing...", initSignal)
		// Utilized Data
		Urls := make([][]string, 0, 0)
		var ContentSignals []<-chan ContentResult
		var IOCompletes []<-chan struct{}
		var Index []int
		// ProgressBar Data
		var Phases []int
		var Signals [][]<-chan struct{}
		var Maximums [][]int
		var Prefix, Postfix [][]string
		for i := 0; i < cnt; i++ {
			if catalogueErrors[i] != nil {
				continue
			}

			urls := []string{}
			for _, catalogue := range catalogues[i] {
				if !catalogue.Fetch {
					urls = append(urls, catalogue.Url)
				}
			}
			if len(urls) == 0 {
				continue
			}

			Index = append(Index, i)
			Urls = append(Urls, urls)
			Maximums = append(Maximums, []int{len(urls), len(urls)})

			hostname, _ := utils.SignatureURL(urls[i])

			Phases = append(Phases, 2)
			Prefix = append(Prefix, []string{outputPrePostfixEachTurn + hostname + " Fetch: ", outputPrePostfixEachTurn + hostname + " Extract: "})
			Postfix = append(Postfix, []string{outputPrePostfixEachTurn, outputPrePostfixEachTurn})

			fetchSignal := make(chan struct{})
			resultChan, extractSignal, ioCompletes := Content(urls, fetchSignal)
			ContentSignals = append(ContentSignals, resultChan)
			IOCompletes = append(IOCompletes, ioCompletes...)
			Signals = append(Signals, []<-chan struct{}{fetchSignal, extractSignal})
		}
		validCnt := len(Index)
		if validCnt == 0 {
			break
		}
		close(initSignal)
		display.ProgressBar(&utils.ProgressBarOption{Writer: writer, Prefix: Prefix, Postfix: Postfix, Maximum: Maximums, Signal: Signals, Phase: Phases})
		hasFail := false
		for i := 0; i < validCnt; i++ {
			result := <-ContentSignals[i]
			k := 0
			index := Index[i]
			for j := 0; j < len(catalogues[index]); j++ {
				if !catalogues[index][j].Fetch {
					if result.Errs[k] == nil {
						catalogues[index][j].Fetch = true
						catalogues[index][j].Content = result.Contents[k]
					} else {
						hasFail = true
					}
					k++
				}
			}
		}
		fmt.Fprintf(writer, "I/O Synchronizing...\n")
		utils.WaitSync(IOCompletes)
		// Here is an early stop to format output.
		if !hasFail {
			break
		} else {
			fmt.Fprintf(writer, outputPrePostfixEachTurn+"Pause %d secs for next turn.\n", pauseSeconds)
			time.Sleep(time.Second * pauseSeconds)
		}
	}
	fmt.Printf("After %dth Turn, finish all pages.\n", times)
	fmt.Printf("Total time: %.0f secs.\n", time.Since(beginTime).Seconds())
	return catalogues, catalogueErrors
}
