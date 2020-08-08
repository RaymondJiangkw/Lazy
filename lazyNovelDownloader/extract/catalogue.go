// catalogue extract the catalogue of novel from its HTML file.
package extract

import (
	"strings"

	"github.com/RaymondJiangkw/Lazy/utils"
	"golang.org/x/net/html"
)

type Chapter struct {
	Name    string
	Url     string
	Content string
	Fetch   bool
}

type Chapters []*Chapter

// Catalogue give the catalogue in the url.
// It takes the method of getting <a> Tags under <dl>.
// However, not all websites use this mechanism. So, there is another
// method of getting the most <a> Tags under a <div> Tag.
func Catalogue(url string) (c Chapters, e error) {
	bodies, errs, ioCompletes := utils.Fetch([]string{url}, &utils.FetchOption{Redirect: true, Refresh: true})
	defer utils.WaitSync(ioCompletes) // Sync I/O here, since `catalogue` only has one page.
	if errs[0] != nil {
		e = errs[0]
		return
	}

	doc, e := html.Parse(strings.NewReader(*bodies[0]))
	if e != nil {
		return
	}

	var aTags []utils.TagA
	if aTags = utils.ParseATags(extractAUnderDL(doc)); len(aTags) > 0 {
		// Method 1
		// Find <a> under <dl>
	} else if aTags = utils.ParseATags(mostAUnderDiv(doc)); len(aTags) > 0 {
		// Method 2
		// Find the most <a> under <div>
	}

	var tmp Chapters // Remove possible head duplications
	var exists map[string]int = make(map[string]int)
	for _, a := range aTags {
		url, err := utils.CompleteURL(url, a.Href)
		if err != nil {
			continue
		}
		tmp = append(tmp, &Chapter{Name: strings.TrimSpace(a.Text), Url: url})
		exists[strings.TrimSpace(a.Text)]++
	}
	// NOTICE: only record the last time it appears.
	for _, t := range tmp {
		// Single
		if exists[t.Name] == 1 {
			c = append(c, t)
		}
		exists[t.Name]--
	}
	return
}

func extractAUnderDL(root *html.Node) []*html.Node {
	return utils.Select(root, "dl a")
}

func mostAUnderDiv(root *html.Node) (ret []*html.Node) {
	tags := utils.Select(root, "div")
	avoidFunc := utils.SelectTagNames([]string{"div"})
	selectFunc := utils.SelectTagNames([]string{"a"})
	for _, tag := range tags {
		c := utils.Extract(tag, selectFunc, &utils.ExtractOption{IsAvoid: avoidFunc})
		if len(c) > len(ret) {
			ret = c
		}
	}
	return
}
