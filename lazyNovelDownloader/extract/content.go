// content extract the content in page.
package extract

import (
	"bufio"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/mvdan/xurls"

	"golang.org/x/net/html"
)

const (
	prefix = "    "
)

func content(body string) (content string, err error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return
	}
	content = mostTextUnderNode("", doc, Candidates([]string{"div"}))
	content = formatString(&content)
	return
}

func Content(info chan<- int, progressFunc func(), urls ...string) (contents []string, errs []error) {
	var signal chan struct{} = make(chan struct{})
	var fetchProgress chan int = make(chan int)
	contents = make([]string, len(urls), len(urls))
	errs = make([]error, len(urls), len(urls))
	go ProgressBar(outputPrefixEachTurn+"Fetch: ", "Fetching...", len(urls), ProgressBarWidth, time.Second, signal, fetchProgress)
	bodies, f_errs := Fetch(false, ioutil.Discard, fetchProgress, urls...)
	signal <- struct{}{}
	var tokens = make(chan struct{}, maximumRoutines)
	var wg sync.WaitGroup
	var finish int
	var lock sync.Mutex
	progressFunc()
	for i, _ := range urls {
		wg.Add(1)
		go func(i int) {
			defer func() {
				lock.Lock()
				finish++
				info <- finish
				lock.Unlock()
				wg.Done()
			}()
			tokens <- struct{}{}
			if f_errs[i] != nil {
				contents[i], errs[i] = "", f_errs[i]
			} else {
				contents[i], errs[i] = content(bodies[i])
			}
			<-tokens
		}(i)
	}
	wg.Wait()
	return
}

func mostTextUnderNode(ret string, n *html.Node, nodes Candidates) string {
	if n.Type == html.ElementNode && nodes.indexOf(n.Data) != -1 {
		text := extractText("", n, nodes, "\n")
		if len(text) > len(ret) {
			ret = text
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ret = mostTextUnderNode(ret, c, nodes)
	}
	return ret
}

func formatString(s *string) (ret string) {
	var trimFunc = func(r rune) bool { return r == '\n' || r == ' ' || r == '\u00a0' || r == '\u3000' }
	rxRelaxed := xurls.Relaxed()
	r := bufio.NewScanner(strings.NewReader(*s))
	for r.Scan() {
		if s := strings.Join(rxRelaxed.Split(strings.TrimFunc(r.Text(), trimFunc), -1), ""); len(s) > 0 {
			ret += prefix + s + "\n"
		}
	}
	return
}
