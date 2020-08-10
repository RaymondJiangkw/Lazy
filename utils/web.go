package utils

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

const (
	redirectKey       = "window.location="
	statusErrorFormat = "Status Code Error %d"
	webCacheFolder    = ".html"
	cacheMaximumSize  = 1024 * 1024 * 256
	fetchMaximumTry   = 5
	pauseSeconds      = 5
	maximumRoutines   = 5
)

// `Fetch`

func NormalizeURL(url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	} else {
		return "http://" + url
	}
}

func CompleteURL(base string, appendix string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(appendix)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(u).String(), nil
}

func PageNameURL(URL string) string {
	base := path.Base(URL)
	pos := strings.Index(base, ".")
	if pos == -1 {
		pos = len(base)
	}
	return base[:pos]
}

// extractKey extract the first key-value pair, which is in the form of key?value? .
func extractKey(key string, content *string) string {
	if loc := strings.Index(*content, key); loc != -1 {
		v := (*content)[loc+len(key):]
		sep := v[:1]
		v = v[1:][:strings.Index(v[1:], sep)]
		return v
	}
	return ""
}

func requestSetHeader(r *http.Request) {
	r.Header.Set("User-Agent", headUserAgent)
	r.Header.Set("Accept", headAccept)
}

// fetch enable the optimization of TIMEOUT Setting, Redirect of `window.onlocation=`, and Cookie Check by double requesting.
// @param url string input will be normalized.
// @param redirect bool determine whether redirect the page, if `window.location=` exists.
func fetch(url string, redirect bool, timeout time.Duration, useCookie bool) (*string, error) {
	url = NormalizeURL(url)
	client := http.Client{
		Timeout: timeout,
	}
	// First `Get` to fetch Cookie
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	requestSetHeader(req)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if useCookie {
		// Add Cookie to Request
		for _, cookie := range resp.Cookies() {
			req.AddCookie(cookie)
		}
		resp.Body.Close()
		// Second `Get` to fetch Content
		resp, err = client.Do(req)
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(statusErrorFormat, resp.StatusCode)
	}
	content, err := DecodeString(resp.Body)
	if err != nil {
		return nil, err
	}
	if redirect {
		if newURL := extractKey(redirectKey, &content); newURL != "" {
			newURL, err = CompleteURL(url, newURL)
			if err != nil {
				return &content, err
			}
			return fetch(newURL, redirect, timeout, useCookie)
		}
	}
	return &content, nil
}

var fetchInitLock sync.Once
var fetchCache *FileCache
var fetchTokens chan struct{}

func initFetch() {
	fetchCache, _ = NewCache(cacheFolder)
	fetchCache.SetCursor(webCacheFolder)
	fetchTokens = make(chan struct{}, maximumRoutines)
}

// @param url string e.g. http://www.google.com www.google.com
func fetchWithTry(url string, redirect bool, errWriter io.Writer, timeout time.Duration, useCookie bool) (data *string, err error) {
	for i := 0; i < fetchMaximumTry; i++ {
		if i != 0 {
			fmt.Fprintf(errWriter, "Encounter Error %v while fetching url %s. Retry the %d th time. Pause %d secs.\n", err, url, i, pauseSeconds)
			time.Sleep(time.Second * pauseSeconds)
		}
		data, err = fetch(url, redirect, timeout, useCookie)
		if err == nil {
			break
		}
	}
	if err != nil {
		fmt.Fprintf(errWriter, "Retry %v Times Out.\n", url)
	}
	return data, err
}

// fetchWriteCache use async IO technique to speed up.
// @param url string Input will be normalized.
func fetchWriteCache(url string, body *string) <-chan struct{} {
	IOComplete := make(chan struct{})
	go func() {
		fetchCache.WriteString("", Id(NormalizeURL(url)), body, false)
		close(IOComplete)
	}()
	return IOComplete
}

// @param url string Input will be normalized.
func fetchReadCache(url string) (*string, error) {
	str, err := fetchCache.ReadString("", Id(NormalizeURL(url)))
	if str == "" && err == nil {
		err = Invalid
	}
	return &str, err
}

// fetchOneURL guarantee returns despite the possibility of broken situation of cache mechanism.
func fetchOneURL(url string, refresh bool, redirect bool, errWriter io.Writer, timeout time.Duration, useCookie bool) (str_ptr *string, err error, c <-chan struct{}) {
	if !refresh {
		str_ptr, err = fetchReadCache(url)
		if err == nil {
			return
		}
	}
	str_ptr, err = fetchWithTry(url, redirect, errWriter, timeout, useCookie)
	if err != nil {
		return nil, err, nil
	}
	// Sync I/O to ensure file has been written after the ending of program.
	c = fetchWriteCache(url, str_ptr)
	return
}

// syncFetch will return after getting all the results of urls, which will be organized in accordion with urls.
// Concurrency-Safe!
func syncFetch(options *FetchOption, urls []string) (data []*string, errs []error, IOCompletes []<-chan struct{}) {
	data = make([]*string, len(urls), len(urls))
	errs = make([]error, len(urls), len(urls))
	IOCompletes = make([]<-chan struct{}, len(urls), len(urls))
	var wg sync.WaitGroup
	for i, url := range urls {
		wg.Add(1)
		go func(i int, url string) {
			defer func() {
				// Signal must be received, and then the task is done.
				if options.Signal != nil {
					options.Signal <- struct{}{}
				}
				wg.Done()
			}()
			fetchTokens <- struct{}{}
			_data, _err, _ch := fetchOneURL(url, options.Refresh, options.Redirect, options.ErrWriter, options.Timeout, options.UseCookie)
			data[i], errs[i], IOCompletes[i] = _data, _err, _ch
			<-fetchTokens
		}(i, url)
	}
	wg.Wait()
	if options.Signal != nil {
		close(options.Signal)
	}
	return
}

// asyncFetch will not wait to return until all the results of urls are gotten.
// Results are sent through receiver whenever one result is ready.
// @returns are useless, but in order to comply with syncFetch, they are used.
// Concurrency-Safe!
func asyncFetch(options *FetchOption, receiver chan<- FetchResult, urls []string) (data []*string, errs []error, IOCompletes []<-chan struct{}) {
	IOCompletes = make([]<-chan struct{}, len(urls), len(urls))
	var wg sync.WaitGroup
	for i, url := range urls {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			fetchTokens <- struct{}{}
			_data, _err, _ch := fetchOneURL(url, options.Refresh, options.Redirect, options.ErrWriter, options.Timeout, options.UseCookie)
			IOCompletes[i] = _ch
			receiver <- FetchResult{data: _data, err: _err, url: url}
			<-fetchTokens
		}(i, url)
	}
	go func() {
		wg.Wait()
		close(receiver)
	}()
	return nil, nil, IOCompletes
}

type FetchResult struct {
	data *string
	err  error
	url  string
}

type FetchOption struct {
	Timeout   time.Duration
	Refresh   bool
	Redirect  bool
	UseCookie bool
	ErrWriter io.Writer
	// signal will be sent whenever a url is processed, either successful or unsuccessful.
	Signal   chan<- struct{}
	Receiver chan<- FetchResult
}

/* Fetch has two types: `async` and `sync`, which is determined by whether {@link options.Receiver} is nil.(nil: `sync`)
 * syncFetch will return after getting all the results of urls, which will be organized in accordion with urls.
 * asyncFetch will not wait to return until all the results of urls are gotten(usually return immediately). Results are sent through receiver whenever one result is ready.
 * Fetch use async IO technique to speed up.
 * @param options *FetchOption
 * @param options.Timeout time.Duration (default: {@link defaultTimeout})
 * @param options.Refresh bool
 * @param options.Redirect bool
 * @param options.ErrWriter io.Writer (default: ioutil.Discard)
 * @param options.Signal chan<-struct{} signal will be sent whenever a url is processed, either successful or unsuccessful.
 * @param options.Receiver chan<-FetchResult, FetchResult:{data *string, err error, url string}.
 */
func Fetch(urls []string, options *FetchOption) ([]*string, []error, []<-chan struct{}) {
	fetchInitLock.Do(initFetch)
	if options == nil {
		options = &FetchOption{}
	}
	if options.ErrWriter == nil {
		options.ErrWriter = ioutil.Discard
	}
	if options.Timeout <= 0 {
		options.Timeout = defaultTimeout
	}
	if options.Receiver == nil {
		return syncFetch(options, urls)
	} else {
		return asyncFetch(options, options.Receiver, urls)
	}
}

// `Extract`

const (
	initNodesNum    = 100
	initRetNodesNum = 10
	dfsSignature    = "DFS"
	bfsSignature    = "BFS"
)

// @param isExpand bool whether expands tag(selected).
func extractDFS(rets []*html.Node, root *html.Node, actedFunc NodeFunc, isAvoid NodeFunc) []*html.Node {
	if actedFunc(root) {
		rets = append(rets, root)
	}
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if isAvoid(c) {
			continue
		}
		rets = extractDFS(rets, c, actedFunc, isAvoid)
	}
	return rets
}

// @param isExpand bool whether expands tag(selected).
func extractBFS(root *html.Node, actedFunc NodeFunc, isAvoid NodeFunc) (rets []*html.Node) {
	var currentNode *html.Node
	var queue []*html.Node
	queue = make([]*html.Node, initNodesNum, initNodesNum)
	rets = make([]*html.Node, initRetNodesNum, initRetNodesNum)
	queue = append(queue, root)
	for len(queue) > 0 {
		currentNode = queue[0]
		rets = queue[1:]
		if actedFunc(currentNode) {
			rets = append(rets, currentNode)
		}
		for c := currentNode.FirstChild; c != nil; c = c.NextSibling {
			if isAvoid(c) {
				continue
			}
			queue = append(queue, c)
		}
	}
	return
}

type ExtractOption struct {
	Type    string // BFS, DFS
	IsAvoid NodeFunc
}

// Extract extract some specific nodes from root *html.Node and avoid some specific nodes.
// @param options.Type string accept {@link dfsSignature} or {@link bfsSignature}. Others will be perceived as "DFS".
// @param options.IsAvoid func(node *html.Node) specify whether needs to avoid tag when expanding. `nil` will be perceived as "always return false" by default.
func Extract(root *html.Node, actedFunc NodeFunc, options *ExtractOption) []*html.Node {
	if options == nil {
		options = &ExtractOption{}
	}
	options.Type = strings.ToUpper(options.Type)
	if options.IsAvoid == nil {
		options.IsAvoid = NodeFunc(func(node *html.Node) bool {
			return false
		})
	}
	switch options.Type {
	case dfsSignature:
		return extractDFS(nil, root, actedFunc, options.IsAvoid)
	case bfsSignature:
		return extractBFS(root, actedFunc, options.IsAvoid)
	default:
		return extractDFS(nil, root, actedFunc, options.IsAvoid)
	}
}

// ExtractText extract texts in node root, except for those under <script> or <style>.
func ExtractText(root *html.Node, sep string, avoidFunc NodeFunc) string {
	nodes := Extract(root, IsTextNode(), &ExtractOption{IsAvoid: SelectTagNames(Candidates([]string{"script", "style"})).Or(avoidFunc)})
	text := ""
	for i, node := range nodes {
		if i > 0 {
			text += sep
		}
		text += node.Data
	}
	return text
}

type Candidates []string

type NodeFunc func(node *html.Node) bool

// indexOf find index of @param str. It will return -1 if not found.
func (c Candidates) indexOf(str string) int {
	for i, s := range c {
		if s == str {
			return i
		}
	}
	return -1
}

func IsElementNode() NodeFunc {
	return NodeFunc(func(node *html.Node) bool {
		return node.Type == html.ElementNode
	})
}

func IsTextNode() NodeFunc {
	return NodeFunc(func(node *html.Node) bool {
		return node.Type == html.TextNode
	})
}

func SelectTagNames(tagNames Candidates) NodeFunc {
	return IsElementNode().And(NodeFunc(func(node *html.Node) bool {
		return tagNames.indexOf(node.Data) != -1
	}))
}

func (f NodeFunc) And(e NodeFunc) NodeFunc {
	if e == nil {
		return f
	}
	return NodeFunc(func(node *html.Node) bool {
		return f(node) && e(node)
	})
}

func (f NodeFunc) Or(e NodeFunc) NodeFunc {
	if e == nil {
		return f
	}
	return NodeFunc(func(node *html.Node) bool {
		return f(node) || e(node)
	})
}

// Selector generate NodeFunc, selecting *html.Node using CSS Selector provided by https://github.com/andybalholm/cascadia.
func Selector(sel string) (NodeFunc, error) {
	Sel, err := cascadia.Parse(sel)
	if err != nil {
		return nil, err
	}
	return NodeFunc(func(node *html.Node) bool {
		return Sel.Match(node)
	}), nil
}

// Select select *html.Node, using CSS Selector provided by https://github.com/andybalholm/cascadia.
func Select(root *html.Node, sel string) []*html.Node {
	Sel, err := cascadia.Parse(sel)
	if err != nil {
		return nil
	}
	return cascadia.QueryAll(root, Sel)
}

type TagA struct {
	Href string
	Text string
}

func (t TagA) String() string {
	return t.Text
}

// ParseATags cannot return []*TagA, in which case data will be recycled.
func ParseATags(n_s []*html.Node) (rets []TagA) {
	for _, n := range n_s {
		if n.Type != html.ElementNode || n.Data != "a" {
			continue
		}
		var tag TagA
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				tag.Href = attr.Val
				break
			}
		}
		tag.Text = strings.TrimSpace(ExtractText(n, "", nil))
		if tag.Href == "" {
			continue
		}
		rets = append(rets, tag)
	}
	return
}

func SignatureURL(URL string) (string, error) {
	URL = NormalizeURL(URL)
	b, e := url.Parse(URL)
	if e != nil {
		return "", e
	}
	return b.Hostname(), nil
}

func AddQueryToURL(URL string, keys []string, values []string) (string, error) {
	if len(keys) != len(values) {
		return "", Invalid
	}
	u, e := url.Parse(URL)
	if e != nil {
		return "", e
	}
	q, _ := url.ParseQuery(u.RawQuery)
	for i := 0; i < len(keys); i++ {
		q.Add(keys[i], values[i])
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
