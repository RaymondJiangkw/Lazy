// web_unsafe extract information from website based on specific rules, which may change from time to time.
// Thus it is not safe.
package utils

import (
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

func search(host string, queryKey string, key string, pageKey string, itemsPerPage int, items int, pageValid func(*string) bool, selector string) (rets []TagA, err error) {
	var getFromOnePage = func(page int) ([]TagA, error) {
		URL, _ := AddQueryToURL(host, []string{queryKey, pageKey}, []string{key, strconv.Itoa(page)})
		bodies, errs, ioCompletes := []*string(nil), []error(nil), []<-chan struct{}(nil)
		defer func() {
			WaitSync(ioCompletes)
		}()
		for {
			var chs []<-chan struct{}
			bodies, errs, chs = Fetch([]string{URL}, &FetchOption{Refresh: true, UseCookie: true})
			ioCompletes = append(ioCompletes, chs...)
			if errs[0] != nil {
				return nil, errs[0]
			}
			if pageValid(bodies[0]) {
				break
			}
			time.Sleep(defaultSleepTime)
		}
		doc, err := html.Parse(strings.NewReader(*bodies[0]))
		if err != nil {
			return nil, err
		}
		nodes := Select(doc, selector)
		return ParseATags(nodes), nil
	}
	i := 0
	for len(rets) < items {
		items, e := getFromOnePage(i * itemsPerPage)
		if e != nil {
			return rets, e
		}
		if len(items) == 0 { // Whenever fails to retrieve any search result, we stop the program.
			return rets, Shortage
		}
		rets = append(rets, items...)
		i++
	}
	return
}

// searchBaidu search result from https://www.baidu.com.
// However, there is an issue that Baidu has strict scraper test, which makes this function cost too much time.
func searchBaidu(key string, items int) ([]TagA, error) {
	const (
		baiduHost         = "https://www.baidu.com/s"
		baiduKey          = "wd"
		baiduPage         = "pn"
		baiduItemsPerPage = 10
	)
	var pageValid = func(content *string) bool {
		return strings.Index(*content, "网络不给力，请稍后重试") == -1
	}
	return search(baiduHost, baiduKey, key, baiduPage, baiduItemsPerPage, items, pageValid, ".t > a")
}

func searchBing(key string, items int) (rets []TagA, err error) {
	const (
		bingHost         = "https://cn.bing.com/search"
		bingKey          = "q"
		bingPage         = "first"
		bingItemsPerPage = 10
	)
	var pageValid = func(content *string) bool {
		return strings.Index(*content, "没有与此相关的结果") == -1
	}
	return search(bingHost, bingKey, key, bingPage, bingItemsPerPage, items, pageValid, "h2 > a")
}

type SearchOption struct {
	Key string
	// accept: baidu, bing. (default: bing)
	Host  string
	Items int // (default:10)
}

func Search(options *SearchOption) ([]TagA, error) {
	if options.Items <= 0 {
		options.Items = defaultItems
	}
	if options.Host == "" {
		options.Host = "bing"
	}
	switch options.Host {
	case "baidu":
		return searchBaidu(options.Key, options.Items)
	default:
		return searchBing(options.Key, options.Items)
	}
}
