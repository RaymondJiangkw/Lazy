// content extract the content in page.
package extract

import (
	"bufio"
	"strings"
	"sync"

	"github.com/RaymondJiangkw/Lazy/utils"

	"github.com/mvdan/xurls"
	"golang.org/x/net/html"
)

const (
	textPrefix = "    "
)

func content(body string) (content string, err error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return
	}
	content = mostTextUnderDiv(doc)
	content = formatString(&content)
	return
}

type ContentResult struct {
	Contents []string
	Errs     []error
}

// Content use async I/O.
// ioCompletes must be wait after receiving from resultChan.
func Content(urls []string, fetchSignal chan<- struct{}) (<-chan ContentResult, <-chan struct{}, []<-chan struct{}) {
	var result ContentResult
	var bodies []*string
	var errs []error
	var ioCompletes []<-chan struct{}
	result.Contents = make([]string, len(urls), len(urls))
	result.Errs = make([]error, len(urls), len(urls))
	resultChan := make(chan ContentResult)
	extractSignal := make(chan struct{})
	tokens := make(chan struct{}, maximumRoutines)
	go func() {
		var wg sync.WaitGroup
		bodies, errs, ioCompletes = utils.Fetch(urls, &utils.FetchOption{Redirect: true, Signal: fetchSignal})
		for i, body := range bodies {
			wg.Add(1)
			go func(i int, body *string) {
				defer func() {
					<-tokens
					extractSignal <- struct{}{}
					wg.Done()
				}()
				tokens <- struct{}{}
				if errs[i] != nil {
					result.Contents[i], result.Errs[i] = "", errs[i]
				} else {
					_content, _err := content(*bodies[i])
					result.Contents[i], result.Errs[i] = _content, _err
				}
				return
			}(i, body)
		}
		wg.Wait()
		close(extractSignal)
		resultChan <- result
		close(resultChan)
	}()
	return resultChan, extractSignal, ioCompletes
}

func mostTextUnderDiv(n *html.Node) string {
	tags := utils.Select(n, "div")
	avoidFunc := utils.SelectTagNames([]string{"div"})
	var ret string
	for _, tag := range tags {
		t := utils.ExtractText(tag, "\n", avoidFunc)
		if len(t) > len(ret) {
			ret = t
		}
	}
	return ret
}

func formatString(s *string) (ret string) {
	var trimFunc = func(r rune) bool { return r == '\n' || r == ' ' || r == '\u00a0' || r == '\u3000' || r == '\t' }
	rxRelaxed := xurls.Relaxed()
	r := bufio.NewScanner(strings.NewReader(*s))
	for r.Scan() {
		if s := strings.Join(rxRelaxed.Split(strings.TrimFunc(r.Text(), trimFunc), -1), ""); len(s) > 0 {
			ret += textPrefix + s + "\n"
		}
	}
	return
}
